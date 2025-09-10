package dao

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

type ShortUrlDAO interface {
	Insert(ctx context.Context, su ShortUrl) error
	FindByShortUrl(ctx context.Context, shortUrl string) (ShortUrl, error)
	FindByShortUrlWithExpired(ctx context.Context, shortUrl string, now int64) (ShortUrl, error)
	FindExpiredList(ctx context.Context, now int64) ([]ShortUrl, error)
	// FindByOriginUrlWithExpired(ctx context.Context, originUrl string, now int64) (ShortUrl, error)
	// FindByOriginUrl(ctx context.Context, originUrl string) (ShortUrl, error)
	DeleteByShortUrl(ctx context.Context, shortUrl string) error
	DeleteExpiredList(ctx context.Context, now int64) ([]string, error)
	Transaction(ctx context.Context, fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error
	WithTransaction(ctx context.Context, fc func(txDAO ShortUrlDAO) error, opts ...*sql.TxOptions) error
}

type ShortUrl struct {
	ShortUrl  string `gorm:"type:char(7) CHARACTER SET ascii COLLATE ascii_bin;not null;primaryKey;column:short_url"`
	OriginUrl string `gorm:"type:varchar(200) CHARACTER SET ascii COLLATE ascii_bin;not null;default '';uniqueIndex:uk_origin_url"`
	ExpiredAt int64  `gorm:"type:bigint;default '-1':index:idx_expired_at"`
}
