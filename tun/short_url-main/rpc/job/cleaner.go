package job

import (
	"context"
	"short_url/rpc/service"
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
	return j.svc.CleanExpired(ctx)
}
