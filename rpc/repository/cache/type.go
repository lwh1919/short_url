package cache

import "context"

type ShortUrlCache interface {
	Get(ctx context.Context, shortUrl string) (originUrl string, err error)
	Set(ctx context.Context, shortUrl string, originUrl string) error
	Del(ctx context.Context, shortUrl string) error
	Refresh(ctx context.Context, shortUrl string) error
}

// BloomFilterCache 布隆过滤器缓存接口
type BloomFilterCache interface {
	// Exist 检查短链接是否可能存在
	Exist(ctx context.Context, shortUrl string) (bool, error)
	// Set 添加短链接到布隆过滤器
	Set(ctx context.Context, shortUrl string) error
	// Rebuild 重建布隆过滤器
	Rebuild(ctx context.Context, shortUrls []string) error
	// GetStats 获取布隆过滤器统计信息
	GetStats(ctx context.Context) (map[string]interface{}, error)
	// IsInitialized 检查布隆过滤器是否已经初始化
	IsInitialized(ctx context.Context) (bool, error)
}
