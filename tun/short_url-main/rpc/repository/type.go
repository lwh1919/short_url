package repository

import "context"

type ShortUrlRepository interface {
	GetOriginUrlByShortUrl(ctx context.Context, shortUrl string) (string, error)
	InsertShortUrl(ctx context.Context, shortUrl, originUrl string) error
	DeleteShortUrlByShortUrl(ctx context.Context, shortUrl string) error
	CleanExpired(ctx context.Context, now int64) error
}
