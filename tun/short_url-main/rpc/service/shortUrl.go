package service

import (
	"context"
	"short_url/pkg/generator"
	"short_url/rpc/repository"
	"time"

	"github.com/to404hanga/pkg404/logger"
)

type CachedShortUrlService struct {
	repo    repository.ShortUrlRepository
	l       logger.Logger
	suffix  string
	Weights []int
}

var _ ShortUrlService = (*CachedShortUrlService)(nil)

func NewCachedShortUrlService(repo repository.ShortUrlRepository, l logger.Logger, suffix string, weights []int) *CachedShortUrlService {
	return &CachedShortUrlService{
		repo:    repo,
		l:       l,
		suffix:  suffix,
		Weights: weights,
	}
}

func (s *CachedShortUrlService) Create(ctx context.Context, originUrl string) (string, error) {
	baseSuffix := ""
	for {
		shortUrl := generator.GenerateShortUrl(originUrl, baseSuffix, s.Weights)
		err := s.repo.InsertShortUrl(ctx, shortUrl, originUrl)
		switch err {
		case nil, repository.ErrUniqueIndexConflict:
			return shortUrl, nil
		case repository.ErrPrimaryKeyConflict:
			baseSuffix += s.suffix
		default:
			return "", err
		}
	}
}

func (s *CachedShortUrlService) Redirect(ctx context.Context, shortUrl string) (string, error) {
	return s.repo.GetOriginUrlByShortUrl(ctx, shortUrl)
}

func (s *CachedShortUrlService) CleanExpired(ctx context.Context) error {
	now := time.Now().Unix()
	return s.repo.CleanExpired(ctx, now)
}
