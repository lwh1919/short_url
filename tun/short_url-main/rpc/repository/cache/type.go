package cache

import "context"

type ShortUrlCache interface {
	Get(ctx context.Context, shortUrl string) (originUrl string, err error)
	Set(ctx context.Context, shortUrl string, originUrl string) error
	Del(ctx context.Context, shortUrl string) error
	Refresh(ctx context.Context, shortUrl string) error
}
