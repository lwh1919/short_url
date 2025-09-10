package main

import (
	"github.com/robfig/cron/v3"
	"github.com/to404hanga/pkg404/grpcx"
)

type App struct {
	GrpcServer *grpcx.Server
	Cron       *cron.Cron
}
