package service

import "context"

type ShortUrlService interface {
	Create(ctx context.Context, originUrl string) (string, error)
	Redirect(ctx context.Context, shortUrl string) (string, error)
	CleanExpired(ctx context.Context) error
}
