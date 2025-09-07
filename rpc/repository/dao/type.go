package dao

import (
	"context"
)

type ShortUrl struct {
	ShortUrl  string `gorm:"type:char(7) CHARACTER SET ascii COLLATE ascii_bin;not null;primaryKey;column:short_url"`
	OriginUrl string `gorm:"type:varchar(200) CHARACTER SET ascii COLLATE ascii_bin;not null;default '';uniqueIndex:uk_origin_url"`
	ExpiredAt int64  `gorm:"type:bigint;default '-1':index:idx_expired_at"`
}

type ShortUrlDAO interface {
	Insert(ctx context.Context, su ShortUrl) error
	FindByShortUrl(ctx context.Context, shortUrl string) (ShortUrl, error)
	FindByShortUrlWithExpired(ctx context.Context, shortUrl string, now int64) (ShortUrl, error)
	FindAllValidShortUrls(ctx context.Context, now int64) ([]ShortUrl, error)

	DeleteByShortUrl(ctx context.Context, shortUrl string) error
	DeleteExpiredList(ctx context.Context, now int64) ([]string, error)
}
