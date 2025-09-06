package job

import (
	"context"
	"short_url_rpc_study/rpc/service"
	"time"
)

type CleanerJob struct {
	svc     service.ShortUrlService
	timeout time.Duration
}

var _ Job = (*CleanerJob)(nil)

func NewCleanerJob(svc service.ShortUrlService, timeout time.Duration) Job {
	return &CleanerJob{
		svc:     svc,
		timeout: timeout,
	}
}

func (j *CleanerJob) Name() string {
	return "cleaner"
}

func (j *CleanerJob) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), j.timeout)
	defer cancel()

	// 清理过期短链接
	if err := j.svc.CleanExpired(ctx); err != nil {
		return err
	}

	// 重建布隆过滤器
	if err := j.svc.RebuildBloomFilter(ctx); err != nil {
		return err
	}

	return nil
}
