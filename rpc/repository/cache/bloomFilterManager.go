package cache

import (
	"context"
	"fmt"
	"short_url/pkg/bloom"
	"sync/atomic"
)

// BloomStats 布隆过滤器统计信息结构体
type BloomStats struct {
	TotalBits         int32
	HashFunctions     int32
	SetBits           int64
	FalsePositiveRate float64
}

// RedisBloomFilterManager Redis布隆过滤器管理器实现
type RedisBloomFilterManager struct {
	bloomService *bloom.BloomService
	key          string
	rebuilding   atomic.Bool // 使用原子标志替代读写锁
}

// 确保实现接口
var _ BloomFilterCache = (*RedisBloomFilterManager)(nil)

// NewRedisBloomFilterManager 创建Redis布隆过滤器管理器
func NewRedisBloomFilterManager(bloomService *bloom.BloomService, key string) BloomFilterCache {
	return &RedisBloomFilterManager{
		bloomService: bloomService,
		key:          key,
	}
}

// Exist 检查短链接是否可能存在
func (r *RedisBloomFilterManager) Exist(ctx context.Context, shortUrl string) (bool, error) {
	return r.bloomService.Exist(ctx, r.key, shortUrl)
}

// Set 添加短链接到布隆过滤器
func (r *RedisBloomFilterManager) Set(ctx context.Context, shortUrl string) error {
	return r.bloomService.Set(ctx, r.key, shortUrl)
}

// Rebuild 重建布隆过滤器
func (r *RedisBloomFilterManager) Rebuild(ctx context.Context, shortUrls []string) error {
	// 使用原子标志避免全程阻塞
	if r.rebuilding.Swap(true) {
		return fmt.Errorf("rebuild already in progress")
	}
	defer r.rebuilding.Store(false)

	// 创建临时key
	tempKey := r.key + "_temp"

	// 清理临时key（如果存在）
	if err := r.bloomService.Clear(ctx, tempKey); err != nil {
		// 记录清理错误但不返回，继续执行
		fmt.Printf("warning: failed to clear temp key %s: %v\n", tempKey, err)
	}

	// 使用pipeline批量写入提高性能
	batchSize := 1000 // 每批处理1000个URL
	for i := 0; i < len(shortUrls); i += batchSize {
		end := i + batchSize
		if end > len(shortUrls) {
			end = len(shortUrls)
		}

		batch := shortUrls[i:end]
		if err := r.bloomService.BatchSet(ctx, tempKey, batch); err != nil {
			// 清理临时key
			if clearErr := r.bloomService.Clear(ctx, tempKey); clearErr != nil {
				fmt.Printf("warning: failed to clear temp key after error: %v\n", clearErr)
			}
			return fmt.Errorf("batch set failed: %w", err)
		}
	}

	// 原子性地替换：直接使用RENAME覆盖旧key，不用先删除后修改
	if err := r.bloomService.Rename(ctx, tempKey, r.key); err != nil {
		// 清理临时key
		if clearErr := r.bloomService.Clear(ctx, tempKey); clearErr != nil {
			fmt.Printf("warning: failed to clear temp key after rename error: %v\n", clearErr)
		}
		return fmt.Errorf("failed to rename bloom filter: %w", err)
	}

	return nil
}

// GetStats 获取布隆过滤器统计信息
func (r *RedisBloomFilterManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return r.bloomService.GetStats(ctx, r.key)
}

// IsInitialized 检查布隆过滤器是否已经初始化
func (r *RedisBloomFilterManager) IsInitialized(ctx context.Context) (bool, error) {
	// 使用类型安全的GetStatsStruct方法
	stats, err := r.bloomService.GetStatsStruct(ctx, r.key)
	if err != nil {
		return false, err
	}

	// 直接访问结构体字段，避免类型断言
	return stats.SetBits > 0, nil
}
