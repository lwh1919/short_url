package cache

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisShortUrlCache struct {
	cmd        redis.Cmdable
	prefix     string
	expiration time.Duration
}

var _ ShortUrlCache = (*RedisShortUrlCache)(nil)

func NewRedisShortUrlCache(cmd redis.Cmdable, prefix string, expiration time.Duration) ShortUrlCache {
	return &RedisShortUrlCache{
		cmd:        cmd,
		prefix:     prefix,
		expiration: expiration,
	}
}

func (r *RedisShortUrlCache) Get(ctx context.Context, shortUrl string) (originUrl string, err error) {
	return r.cmd.Get(ctx, r.key(shortUrl)).Result()
}

func (r *RedisShortUrlCache) Set(ctx context.Context, shortUrl string, originUrl string) error {
	_, err := r.cmd.Set(ctx, r.key(shortUrl), originUrl, r.expiration+time.Duration(rand.IntN(7201)-3600)).Result() // 随机加减一小时过期时间
	return err
}

func (r *RedisShortUrlCache) Del(ctx context.Context, shortUrl string) error {
	return r.cmd.Del(ctx, r.key(shortUrl)).Err()
}

func (r *RedisShortUrlCache) Refresh(ctx context.Context, shortUrl string) error {
	return r.cmd.Expire(ctx, r.key(shortUrl), r.expiration+time.Duration(rand.IntN(7201)-3600)).Err() // 随机加减一小时过期时间
}

func (r *RedisShortUrlCache) key(shortUrl string) string {
	return r.prefix + ":" + shortUrl
}
