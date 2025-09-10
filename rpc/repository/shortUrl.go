package repository

import (
	"context"
	"fmt"
	"math/rand/v2"
	"short_url/rpc/repository/cache"
	"short_url/rpc/repository/dao"
	"time"

	"github.com/to404hanga/pkg404/cachex/lru"
	"github.com/to404hanga/pkg404/logger"
	"golang.org/x/sync/singleflight"
)

type CachedShortUrlRepository struct {
	lru           *lru.Cache
	lruExpiration time.Duration
	cache         cache.ShortUrlCache
	bloomFilter   cache.BloomFilterCache
	dao           dao.ShortUrlDAO
	l             logger.Logger
	requestGroup  singleflight.Group
}

type lruItem struct {
	originUrl string
	expiredAt int64
}

var _ ShortUrlRepository = (*CachedShortUrlRepository)(nil)

var (
	ErrPrimaryKeyConflict  = dao.ErrPrimaryKeyConflict
	ErrUniqueIndexConflict = dao.ErrUniqueIndexConflict
)

func NewCachedShortUrlRepository(lruSize int, lruExpiration time.Duration, cache cache.ShortUrlCache, bloomFilter cache.BloomFilterCache, dao dao.ShortUrlDAO, l logger.Logger) ShortUrlRepository {
	lru, err := lru.New(lruSize)
	if err != nil {
		panic(err)
	}
	return &CachedShortUrlRepository{
		lru:           lru,
		lruExpiration: lruExpiration,
		cache:         cache,
		bloomFilter:   bloomFilter,
		dao:           dao,
		l:             l,
		requestGroup:  singleflight.Group{},
	}

}

func (c *CachedShortUrlRepository) GetOriginUrlByShortUrl(ctx context.Context, shortUrl string) (string, error) {
	now := time.Now().Unix()

	result, err, _ := c.requestGroup.Do("lru_redis_"+shortUrl, func() (interface{}, error) {
		// 先查本地缓存，若本地缓存存在直接返回
		val, ok := c.lru.Get(shortUrl)
		if ok {
			if item, ok := val.(lruItem); ok && item.expiredAt >= now {
				return item.originUrl, nil
			}
		}

		// 若本地缓存不存在，从 redis 读取并更新本地缓存
		originUrl, err := c.cache.Get(ctx, shortUrl)
		if err == nil {
			go func() {
				newCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				if err := c.cache.Refresh(newCtx, shortUrl); err == nil {
					c.l.Error("failed to refresh redis cache",
						logger.Error(err),
						logger.String("short_url", shortUrl),
					)
				}
			}()

			c.lru.Add(shortUrl, lruItem{
				originUrl: originUrl,
				expiredAt: int64(time.Now().Add(time.Duration(c.lruExpiration.Seconds()+float64(rand.IntN(7201)-3600)) * time.Second).Unix()),
			})

			return originUrl, err
		}
		c.l.Error("cache.Get failed",
			logger.Error(err),
			logger.String("short_url", shortUrl),
		)

		// 在查询数据库之前，先检查布隆过滤器
		c.l.Info("布隆过滤器运作")
		initialized, err := c.bloomFilter.IsInitialized(ctx)
		if err != nil {
			c.l.Error("failed to check bloom filter initialization",
				logger.Error(err),
				logger.String("short_url", shortUrl),
			)
			// 无法检查初始化状态，继续查询数据库（降级处理）
			c.l.Warn("falling back to database query due to bloom filter initialization check failure",
				logger.String("short_url", shortUrl),
			)
		} else if !initialized {
			// 布隆过滤器未初始化，跳过布隆过滤器检查，直接查询数据库
			c.l.Warn("bloom filter not initialized, skipping bloom filter check",
				logger.String("short_url", shortUrl),
			)
		} else {
			// 布隆过滤器已初始化，进行正常的布隆过滤器检查
			exists, err := c.bloomFilter.Exist(ctx, shortUrl)
			if err != nil {
				c.l.Error("bloom filter check failed",
					logger.Error(err),
					logger.String("short_url", shortUrl),
				)
				// 布隆过滤器检查失败，继续查询数据库（降级处理）
				c.l.Warn("falling back to database query due to bloom filter failure",
					logger.String("short_url", shortUrl),
				)
			} else if !exists {
				// 布隆过滤器显示短链接不存在，直接返回错误
				// 注意：这里可能存在假阳性，但为了性能考虑，我们信任布隆过滤器的结果
				return "", fmt.Errorf("short url not found: %s", shortUrl)
			}
		}

		// 若 redis 读取失败，从数据库读取并更新本地 lru 缓存和 redis 缓存
		su, err := c.dao.FindByShortUrlWithExpired(ctx, shortUrl, now)
		fmt.Println("查询数据库")
		if err != nil {
			return "", err
		}
		fmt.Println(c.bloomFilter.Exist(ctx, shortUrl))
		go func() {
			newCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// 异步更新 redis 缓存
			if err = c.cache.Set(newCtx, shortUrl, su.OriginUrl); err != nil {
				c.l.Error("failed to set redis cache",
					logger.Error(err),
					logger.String("short_url", shortUrl),
					logger.String("origin_url", su.OriginUrl),
				)
			}
		}()
		// 同步更新本地 lru 缓存
		c.lru.Add(shortUrl, lruItem{
			originUrl: su.OriginUrl,
			expiredAt: int64(time.Now().Add(time.Duration(c.lruExpiration.Seconds()+float64(rand.IntN(7201)-3600)) * time.Second).Unix()),
		})

		return su.OriginUrl, nil
	})
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

func (c *CachedShortUrlRepository) InsertShortUrl(ctx context.Context, shortUrl, originUrl string) error {

	// 插入数据库
	err := c.dao.Insert(ctx, dao.ShortUrl{
		ShortUrl:  shortUrl,
		OriginUrl: originUrl,
		ExpiredAt: time.Now().AddDate(1, 0, 0).Unix(), // 有效期一年
	})
	if err != nil {
		return err
	}
	//每一条都开协程去处理太麻烦了，这里应该优化为协程池消息处理，但是暂时注释，先进性测试
	//异步添加到布隆过滤器
	go func() {
		// 使用独立上下文执行异步布隆过滤器更新，避免被请求上下文取消
		newCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := c.bloomFilter.Set(newCtx, shortUrl); err != nil {
			c.l.Error("failed to add to bloom filter",
				logger.Error(err),
				logger.String("short_url", shortUrl),
			)
		}
	}()

	return nil
}

func (c *CachedShortUrlRepository) DeleteShortUrlByShortUrl(ctx context.Context, shortUrl string) error {
	err := c.dao.DeleteByShortUrl(ctx, shortUrl)
	if err == nil {
		go func() {
			newCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			// 异步删除 redis 缓存
			if err = c.cache.Del(newCtx, shortUrl); err != nil {
				c.l.Error("failed to delete redis cache",
					logger.Error(err),
					logger.String("short_url", shortUrl),
				)
			}
		}()
		// 同步删除本地 lru 缓存
		c.lru.Remove(shortUrl)
	}
	return err
}

func (c *CachedShortUrlRepository) CleanExpired(ctx context.Context, now int64) error {
	deleteList, err := c.dao.DeleteExpiredList(ctx, now)
	if err == nil {
		newCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		go func() {
			defer cancel()
			for _, shortUrl := range deleteList {
				// 异步删除 redis 缓存
				if err = c.cache.Del(newCtx, shortUrl); err != nil {
					c.l.Error("failed to delete redis cache",
						logger.Error(err),
						logger.String("short_url", shortUrl),
					)
				}
				// 异步删除本地 lru 缓存
				c.lru.Remove(shortUrl)
			}
		}()
	}
	return err
}

// 重建布隆过滤器
func (c *CachedShortUrlRepository) RebuildBloomFilter(ctx context.Context) error {
	// 获取所有未过期的短链接
	shortUrls, err := c.dao.FindAllValidShortUrls(ctx, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to get valid short urls: %w", err)
	}

	// 提取短链接字符串
	shortUrlList := make([]string, 0, len(shortUrls))
	for _, su := range shortUrls {
		shortUrlList = append(shortUrlList, su.ShortUrl)
	}

	// 重建布隆过滤器
	if err := c.bloomFilter.Rebuild(ctx, shortUrlList); err != nil {
		return fmt.Errorf("failed to rebuild bloom filter: %w", err)
	}

	c.l.Info("bloom filter rebuilt successfully",
		logger.Int("total_short_urls", len(shortUrlList)),
	)

	return nil
}
