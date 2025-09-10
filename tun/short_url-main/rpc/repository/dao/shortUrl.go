package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"short_url/pkg/generator"
	"sync"
	"time"

	"github.com/to404hanga/pkg404/logger"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormShortUrlDAO struct {
	db            *gorm.DB
	l             logger.Logger
	buffer        []ShortUrl      // 环形缓冲区
	bufferSize    int             // 缓冲区大小
	readPos       int             // 读取位置
	writePos      int             // 写入位置
	batchSize     int             // 批量大小
	flushInterval time.Duration   // 刷新间隔
	wg            sync.WaitGroup  // 用于等待worker完成
	stopChan      chan struct{}   // 用于停止worker
	flushPool     sync.Pool       // 用于批量处理的slice池
	flushChan     chan []ShortUrl // 用于异步刷新
	mu            sync.Mutex      // 保护缓冲区访问
}

func (g *GormShortUrlDAO) batchWorker() {
	defer g.wg.Done()

	ticker := time.NewTicker(g.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopChan:
			// 处理剩余请求
			g.flushRemaining()
			return

		case <-ticker.C:
			g.flushIfNeeded()
		}
	}
}

func (g *GormShortUrlDAO) flushIfNeeded() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.readPos == g.writePos {
		return
	}

	batch := g.getBatch()
	if len(batch) > 0 {
		g.flushChan <- batch
	}
}

func (g *GormShortUrlDAO) flushRemaining() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for g.readPos != g.writePos {
		batch := g.getBatch()
		if len(batch) > 0 {
			g.flushChan <- batch
		}
	}
}

func (g *GormShortUrlDAO) getBatch() []ShortUrl {
	batch := g.flushPool.Get().([]ShortUrl)
	batch = batch[:0]

	for g.readPos != g.writePos && len(batch) < g.batchSize {
		batch = append(batch, g.buffer[g.readPos])
		g.readPos = (g.readPos + 1) % g.bufferSize
	}

	return batch
}

func (g *GormShortUrlDAO) flushBatch(ctx context.Context, batch []ShortUrl) {
	defer g.flushPool.Put(batch)

	// 按表名分组
	groups := make(map[string][]ShortUrl)
	for _, su := range batch {
		table := g.tableName(su.ShortUrl)
		groups[table] = append(groups[table], su)
	}

	// 使用worker池处理每个表
	var wg sync.WaitGroup
	wg.Add(len(groups))

	for table, sus := range groups {
		go func(table string, sus []ShortUrl) {
			defer wg.Done()
			g.db.WithContext(ctx).Table(table).Transaction(func(tx *gorm.DB) error {
				result := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "short_url"}},
					DoUpdates: clause.Assignments(map[string]any{}),
				}).Create(&sus)

				if result.RowsAffected != int64(len(sus)) {
					// 处理冲突
					for i := result.RowsAffected; i < int64(len(sus)); i++ {
						var existing ShortUrl
						if err := tx.Where("short_url = ?", sus[i].ShortUrl).First(&existing).Error; err != nil {
							g.l.Error("batch insert conflict check failed",
								logger.Error(err),
								logger.String("short_url", sus[i].ShortUrl))
							continue
						}
						if existing.OriginUrl != sus[i].OriginUrl {
							g.l.Warn("primary key conflict detected",
								logger.String("short_url", sus[i].ShortUrl),
								logger.String("existing_origin_url", existing.OriginUrl),
								logger.String("new_origin_url", sus[i].OriginUrl))
						}
					}
				}
				return nil
			})
		}(table, sus)
	}

	wg.Wait()
}

func (g *GormShortUrlDAO) Close() error {
	close(g.stopChan)
	g.wg.Wait()
	return nil
}

var _ ShortUrlDAO = (*GormShortUrlDAO)(nil)

var (
	ErrPrimaryKeyConflict  = errors.New("primary key conflict")
	ErrUniqueIndexConflict = errors.New("unique index conflict")
	ErrDataNotFound        = gorm.ErrRecordNotFound
)

func NewGormShortUrlDAO(db *gorm.DB, l logger.Logger) ShortUrlDAO {
	dao := &GormShortUrlDAO{
		db:            db,
		l:             l,
		buffer:        make([]ShortUrl, 2000), // 双倍大小以提供缓冲
		bufferSize:    2000,
		batchSize:     1000,
		flushInterval: 50 * time.Millisecond,
		stopChan:      make(chan struct{}),
		flushChan:     make(chan []ShortUrl, 10), // 10个并发刷新
	}

	// 初始化slice池
	dao.flushPool.New = func() interface{} {
		return make([]ShortUrl, 0, dao.batchSize)
	}

	// 启动批量处理goroutine
	dao.wg.Add(1)
	go dao.batchWorker()

	// 启动异步刷新worker
	for i := 0; i < 5; i++ {
		go func() {
			for batch := range dao.flushChan {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				dao.flushBatch(ctx, batch)
				cancel()
			}
		}()
	}

	return dao
}

func (g *GormShortUrlDAO) tableName(shortUrlOrSuffix string) string {
	if len(shortUrlOrSuffix) == 1 {
		return "short_url_" + shortUrlOrSuffix
	}
	return fmt.Sprintf("short_url_%s", string(shortUrlOrSuffix[0]))
}

func (g *GormShortUrlDAO) Insert(ctx context.Context, su ShortUrl) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	nextPos := (g.writePos + 1) % g.bufferSize
	if nextPos == g.readPos {
		return errors.New("buffer full")
	}

	g.buffer[g.writePos] = su
	g.writePos = nextPos

	// 如果达到批量大小，立即刷新
	if (g.writePos-g.readPos+g.bufferSize)%g.bufferSize >= g.batchSize {
		g.flushIfNeeded()
	}

	return nil
}

/*
// 原单次插入实现，保留作为参考
func (g *GormShortUrlDAO) Insert(ctx context.Context, su ShortUrl) error {
	tableName := g.tableName(su.ShortUrl)
	err := g.db.WithContext(ctx).Table(tableName).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "short_url"}}, // 唯一索引列
			DoUpdates: clause.Assignments(map[string]any{}),
		}).Create(&su)

		if result.Error != nil {
			return result.Error
		}

		// 通过 RowsAffected 判断实际操作
		if result.RowsAffected == 0 {
			// 冲突发生后的处理逻辑
			var existing ShortUrl
			if err := tx.Where("short_url = ?", su.ShortUrl).First(&existing).Error; err != nil {
				return err
			}
			if existing.OriginUrl != su.OriginUrl {
				return ErrPrimaryKeyConflict
			}
			return ErrUniqueIndexConflict
		}
		return nil
	})
	return err
}
*/

func (g *GormShortUrlDAO) FindByShortUrlWithExpired(ctx context.Context, shortUrl string, now int64) (ShortUrl, error) {
	var su ShortUrl
	err := g.db.WithContext(ctx).Table(g.tableName(shortUrl)).Where("short_url = ?", shortUrl).Where("expired_at > ?", now).First(&su).Error
	return su, err
}

func (g *GormShortUrlDAO) FindByShortUrl(ctx context.Context, shortUrl string) (ShortUrl, error) {
	var su ShortUrl
	err := g.db.WithContext(ctx).Table(g.tableName(shortUrl)).Where("short_url = ?", shortUrl).First(&su).Error
	return su, err
}

func (g *GormShortUrlDAO) FindByOriginUrlWithExpired(ctx context.Context, originUrl string, now int64) (ShortUrl, error) {
	var su ShortUrl
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(62)
	for i := 0; i < 62; i++ {
		go func(internalCtx context.Context, suffix string) {
			defer wg.Done()
			select {
			case <-internalCtx.Done():
				return
			default:
				var internalSu ShortUrl
				if err := g.db.WithContext(internalCtx).
					Table(g.tableName(suffix)).
					Where("origin_url = ?", originUrl).
					Where("expired_at > ?", now).
					First(&internalSu).Error; err == nil {
					su = internalSu
					cancel()
				}
			}
		}(newCtx, string(generator.BASE62CHARSET[i]))
	}
	wg.Wait()
	if su.ShortUrl == "" {
		return ShortUrl{}, ErrDataNotFound
	}
	return su, nil
}

func (g *GormShortUrlDAO) FindByOriginUrlWithExpiredV1(ctx context.Context, originUrl string, now int64) (ShortUrl, error) {
	var (
		su   ShortUrl
		lock sync.Mutex
	)
	g.executeUnshardedQuery(ctx, func(iCtx context.Context, suffix string, db *gorm.DB) error {
		var internalSu ShortUrl
		if err := db.WithContext(iCtx).
			Table(g.tableName(suffix)).
			Where("origin_url =?", originUrl).
			Where("expired_at >?", now).
			First(&internalSu).Error; err != nil {
			g.l.Error("FindByOriginUrlWithExpiredV1 failed",
				logger.Error(err),
				logger.String("suffix", suffix),
				logger.String("origin_url", originUrl),
				logger.Int64("expired_at", now),
			)
			return err
		}
		lock.Lock()
		su = internalSu
		lock.Unlock()
		return nil
	})
	if su.ShortUrl == "" {
		return ShortUrl{}, ErrDataNotFound
	}
	return su, nil
}

func (g *GormShortUrlDAO) FindByOriginUrl(ctx context.Context, originUrl string) (ShortUrl, error) {
	var su ShortUrl
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(62)
	for i := 0; i < 62; i++ {
		go func(internalCtx context.Context, suffix string) {
			defer wg.Done()
			select {
			case <-internalCtx.Done():
				return
			default:
				var internalSu ShortUrl
				if err := g.db.WithContext(internalCtx).
					Table(g.tableName(suffix)).
					Where("origin_url = ?", originUrl).
					First(&internalSu).Error; err == nil {
					su = internalSu
					cancel()
				}
			}
		}(newCtx, string(generator.BASE62CHARSET[i]))
	}
	wg.Wait()
	if su.ShortUrl == "" {
		return ShortUrl{}, ErrDataNotFound
	}
	return su, nil
}

func (g *GormShortUrlDAO) FindByOriginUrlV1(ctx context.Context, originUrl string) (ShortUrl, error) {
	var (
		su   ShortUrl
		lock sync.Mutex
	)
	g.executeUnshardedQuery(ctx, func(iCtx context.Context, suffix string, db *gorm.DB) error {
		var internalSu ShortUrl
		if err := db.WithContext(iCtx).
			Table(g.tableName(suffix)).
			Where("origin_url =?", originUrl).
			First(&internalSu).Error; err != nil {
			g.l.Error("FindByOriginUrlV1 failed",
				logger.Error(err),
				logger.String("suffix", suffix),
				logger.String("origin_url", originUrl),
			)
			return err
		}
		lock.Lock()
		su = internalSu
		lock.Unlock()
		return nil
	})
	if su.ShortUrl == "" {
		return ShortUrl{}, ErrDataNotFound
	}
	return su, nil
}

func (g *GormShortUrlDAO) FindExpiredList(ctx context.Context, now int64) ([]ShortUrl, error) {
	var (
		sus  []ShortUrl
		wg   sync.WaitGroup
		lock sync.Mutex
	)
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	wg.Add(62)
	for i := 0; i < 62; i++ {
		go func(internalCtx context.Context, suffix string) {
			defer wg.Done()
			select {
			case <-internalCtx.Done():
				return
			default:
				var internalSus []ShortUrl
				if err := g.db.WithContext(internalCtx).
					Table(g.tableName(suffix)).
					Where("expired_at <=?", now).
					Find(&internalSus).Error; err == nil {
					lock.Lock()
					sus = append(sus, internalSus...)
					lock.Unlock()
					cancel()
				}
			}
		}(newCtx, string(generator.BASE62CHARSET[i]))
	}
	wg.Wait()
	if len(sus) == 0 {
		return nil, ErrDataNotFound
	}
	return sus, nil
}

func (g *GormShortUrlDAO) FindExpiredListV1(ctx context.Context, now int64) ([]ShortUrl, error) {
	var (
		sus  []ShortUrl
		lock sync.Mutex
	)
	g.executeUnshardedQuery(ctx, func(iCtx context.Context, suffix string, db *gorm.DB) error {
		var internalSus []ShortUrl
		err := db.WithContext(iCtx).
			Table(g.tableName(suffix)).
			Where("expired_at <=?", now).
			Find(&internalSus).Error
		if err != nil {
			g.l.Error("FindExpiredListV1 failed",
				logger.Error(err),
				logger.String("suffix", suffix),
				logger.Int64("expired_at", now),
			)
			return err
		}
		lock.Lock()
		sus = append(sus, internalSus...)
		lock.Unlock()
		return nil
	})
	if len(sus) == 0 {
		return nil, ErrDataNotFound
	}
	return sus, nil
}

// 批量执行不分表操作的抽象方法
func (g *GormShortUrlDAO) executeUnshardedQuery(ctx context.Context, fn func(iCtx context.Context, suffix string, db *gorm.DB) error) {
	var wg sync.WaitGroup
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	wg.Add(62)
	for i := 0; i < 62; i++ {
		go func(internalCtx context.Context, suffix string) {
			defer wg.Done()
			select {
			case <-internalCtx.Done():
				return
			default:
				if err := fn(internalCtx, suffix, g.db); err == nil {
					cancel()
				}
			}
		}(newCtx, string(generator.BASE62CHARSET[i]))
	}
	wg.Wait()
}

func (g *GormShortUrlDAO) DeleteByShortUrl(ctx context.Context, shortUrl string) error {
	return g.db.WithContext(ctx).Table(g.tableName(shortUrl)).Where("short_url = ?", shortUrl).Delete(&ShortUrl{}).Error
}

func (g *GormShortUrlDAO) DeleteExpiredList(ctx context.Context, now int64) ([]string, error) {
	var (
		retList []string
		group   errgroup.Group
		lock    sync.Mutex
	)
	for i := 0; i < 62; i++ {
		group.Go(func() error {
			tableName := "short_url_" + string(generator.BASE62CHARSET[i])
			for {
				var ret []string
				// 查询可删除列表
				err := g.db.WithContext(ctx).Table(tableName).Select("short_url").
					Where("expired_at < ?", now).Order("expired_at ASC").Limit(100).
					Find(&ret).Error
				if err != nil {
					return err
				}
				if len(ret) == 0 {
					break // 无更多数据可删除
				}
				err = g.db.WithContext(ctx).Table(tableName).Where("short_url IN ?", ret).Delete(&ShortUrl{}).Error
				if err != nil {
					return err
				}

				lock.Lock()
				retList = append(retList, ret...)
				lock.Unlock()

				time.Sleep(100 * time.Millisecond) // 避免高频操作压垮数据库
			}
			return nil
		})
	}
	return retList, group.Wait()
}

func (g *GormShortUrlDAO) Transaction(ctx context.Context, fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fc(tx)
	}, opts...)
}

func (g *GormShortUrlDAO) WithTransaction(ctx context.Context, fc func(txDAO ShortUrlDAO) error, opts ...*sql.TxOptions) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDAO := &GormShortUrlDAO{db: tx}
		return fc(txDAO)
	}, opts...)
}
