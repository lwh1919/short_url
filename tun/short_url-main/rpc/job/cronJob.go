package job

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/logger"
)

type CronJobBuilder struct {
	l logger.Logger
}

func NewCronJobBuilder(l logger.Logger) *CronJobBuilder {
	return &CronJobBuilder{
		l: l,
	}
}

func (b *CronJobBuilder) Build(job Job) cron.Job {
	name := job.Name()
	return cronJobAdapterFunc(func() {
		if viper.GetBool("job.enabled") {
			start := time.Now()
			b.l.Debug("Job started", logger.String("name", name))
			err := job.Run()
			if err != nil {
				b.l.Error("Job failed", logger.String("name", name), logger.Error(err))
			} else {
				b.l.Debug("Job finished", logger.String("name", name))
			}
			duration := time.Since(start).Milliseconds()
			b.l.Info("Job duration", logger.String("name", name), logger.Int64("duration_ms", duration))
		} else {
			b.l.Debug("Job disabled", logger.String("name", name))
		}
	})
}

type cronJobAdapterFunc func()

func (f cronJobAdapterFunc) Run() {
	f()
}
