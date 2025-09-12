package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/to404hanga/pkg404/logger"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"runtime"
	"short_url/pkg/generator"
	"sync"
	"time"
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
	antsPool      *ants.Pool      // ants协程池
	closed        bool            // 标记是否已关闭
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
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil
	}
	g.closed = true
	g.mu.Unlock()

	// 1. 停止batch worker
	close(g.stopChan)

	// 2. 等待batch worker完成
	g.wg.Wait()

	// 3. 关闭flush channel
	close(g.flushChan)

	// 4. 等待ants协程池中的任务完成并释放资源
	g.antsPool.Release()

	g.l.Info("GormShortUrlDAO closed successfully")
	return nil
}

var _ ShortUrlDAO = (*GormShortUrlDAO)(nil)

var (
	ErrPrimaryKeyConflict  = errors.New("primary key conflict")
	ErrUniqueIndexConflict = errors.New("unique index conflict")
	ErrDataNotFound        = gorm.ErrRecordNotFound
)

func NewGormShortUrlDAO(db *gorm.DB, l logger.Logger) ShortUrlDAO {
	// 计算协程池大小：I/O密集型任务，使用CPU核心数的3倍
	cpuCount := runtime.NumCPU()
	poolSize := cpuCount * 3
	if poolSize < 4 {
		poolSize = 4 // 最小保证4个协程
	}

	// 创建ants协程池
	antsPool, err := ants.NewPool(poolSize,
		ants.WithPreAlloc(true),         // 预分配内存，提高性能
		ants.WithMaxBlockingTasks(1000), // 最大阻塞任务数
		ants.WithPanicHandler(func(p interface{}) {
			l.Error("Ants pool panic recovered", logger.Any("panic", p))
		}),
	)
	if err != nil {
		l.Error("Failed to create ants pool", logger.Error(err))
		panic(err)
	}

	flushChanBuffer := 10
	dao := &GormShortUrlDAO{
		db:            db,
		l:             l,
		buffer:        make([]ShortUrl, 2000), // 双倍大小以提供缓冲
		bufferSize:    2000,
		batchSize:     1000,
		flushInterval: 50 * time.Millisecond,
		stopChan:      make(chan struct{}),
		antsPool:      antsPool,
		flushChan:     make(chan []ShortUrl, flushChanBuffer),
		closed:        false,
	}

	// 初始化slice池
	dao.flushPool.New = func() interface{} {
		return make([]ShortUrl, 0, dao.batchSize)
	}

	// 启动batch worker
	dao.wg.Add(1)
	go dao.batchWorker()

	// 启动batch消费者goroutine - 串行消费，并发执行
	dao.wg.Add(1)
	go func() {
		defer dao.wg.Done()
		for batch := range dao.flushChan {
			// 使用ants协程池处理batch，正确处理错误
			batchCopy := batch // 避免闭包问题
			err := dao.antsPool.Submit(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				dao.flushBatch(ctx, batchCopy)
			})
			if err != nil {
				// 如果协程池提交失败，同步执行以避免数据丢失
				l.Error("Failed to submit batch to ants pool, executing synchronously",
					logger.Error(err),
					logger.Int("batch_size", len(batchCopy)))
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				dao.flushBatch(ctx, batchCopy)
				cancel()
			}
		}
	}()

	// 记录协程池大小
	l.Info("ShortUrlDAO initialized with ants pool and channel",
		logger.Int("cpu_count", cpuCount),
		logger.Int("pool_size", poolSize),
		logger.Int("buffer_size", dao.bufferSize),
		logger.Int("batch_size", dao.batchSize),
		logger.Int("flush_chan_buffer", flushChanBuffer))

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

// FindAllValidShortUrls 获取所有未过期的短链接
func (g *GormShortUrlDAO) FindAllValidShortUrls(ctx context.Context, now int64) ([]ShortUrl, error) {
	var (
		sus  []ShortUrl
		lock sync.Mutex
	)

	g.executeUnshardedQuery(ctx, func(iCtx context.Context, suffix string, db *gorm.DB) error {
		var internalSus []ShortUrl
		err := db.WithContext(iCtx).
			Table(g.tableName(suffix)).
			Where("expired_at > ?", now).
			Find(&internalSus).Error
		if err != nil {
			g.l.Error("FindAllValidShortUrls failed",
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

	return sus, nil
}
func (g *GormShortUrlDAO) WithTransaction(ctx context.Context, fc func(txDAO ShortUrlDAO) error, opts ...*sql.TxOptions) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务中的DAO不需要ants协程池，直接使用同步操作
		txDAO := &GormShortUrlDAO{
			db: tx,
			l:  g.l,
		}
		return fc(txDAO)
	}, opts...)
}
