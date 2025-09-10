package ioc

import (
	"short_url/rpc/job"
	"short_url/rpc/service"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/logger"
)

func InitCleanerJob(svc service.ShortUrlService) job.Job {
	timeout := viper.GetInt("job.timeout")
	if timeout <= 0 {
		panic("job.timeout must be positive")
	}
	return job.NewCleanerJob(svc, time.Duration(timeout)*time.Second)
}

func InitJobs(l logger.Logger, j job.Job) *cron.Cron {
	expr := viper.GetString("job.expr")
	if expr == "" {
		panic("job.expr must be set")
	}
	builder := job.NewCronJobBuilder(l)
	c := cron.New(cron.WithSeconds())
	if _, err := c.AddJob(expr, builder.Build(j)); err != nil {
		panic(err)
	}
	return c
}
