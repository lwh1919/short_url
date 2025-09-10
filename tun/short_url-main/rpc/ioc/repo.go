package ioc

import (
	"short_url/rpc/repository"
	"short_url/rpc/repository/cache"
	"short_url/rpc/repository/dao"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/logger"
)

func InitCachedRepository(cache cache.ShortUrlCache, dao dao.ShortUrlDAO, l logger.Logger) repository.ShortUrlRepository {
	type Config struct {
		Size       int     `yaml:"size"`
		Percentage float64 `yaml:"percentage"`
		Expiration int     `yaml:"expiration"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("lru", &cfg); err != nil {
		panic(err)
	}

	// 如果 size 值非法或未提供，根据 percentage 确定 size 的值
	if cfg.Size <= 0 {
		v, err := mem.VirtualMemory()
		if err != nil {
			panic(err)
		}

		/*
		 * 1. 关于 /256 的解释
		 *    我们认为使用 lru 缓存时，每一对 (短链接, 原链接) 的键值对所占内存约为 256 字节，
		 *    短链接和原链接均采用 ascii 编码，
		 *    短链接长度为 7，原链接最大长度为 200
		 * 2. 关于 cfg.Percentage 的值的选取
		 *    一般为 1% ~ 5%，可取中间值 3%
		 */
		cfg.Size = int(float64(v.Total) / 100 * cfg.Percentage / 256)
	}

	expiration := time.Duration(cfg.Expiration) * time.Second
	return repository.NewCachedShortUrlRepository(cfg.Size, expiration, cache, dao, l)
}
