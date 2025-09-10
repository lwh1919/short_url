package repository

import (
	"context"
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

func NewCachedShortUrlRepository(lruSize int, lruExpiration time.Duration, cache cache.ShortUrlCache, dao dao.ShortUrlDAO, l logger.Logger) ShortUrlRepository {
	lru, err := lru.New(lruSize)
	if err != nil {
		panic(err)
	}
	return &CachedShortUrlRepository{
		lru:           lru,
		lruExpiration: lruExpiration,
		cache:         cache,
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
				newCtx, cancel := context.WithTimeout(ctx, time.Second)
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

		// 若 redis 读取失败，从数据库读取并更新本地 lru 缓存和 redis 缓存
		su, err := c.dao.FindByShortUrlWithExpired(ctx, shortUrl, now)
		if err != nil {
			return "", err
		}

		go func() {
			newCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
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
	return c.dao.Insert(ctx, dao.ShortUrl{
		ShortUrl:  shortUrl,
		OriginUrl: originUrl,
		ExpiredAt: time.Now().AddDate(1, 0, 0).Unix(), // 有效期一年
	})
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
