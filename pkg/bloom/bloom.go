package bloom

import (
	"context"
	"fmt"
	"math"

	"github.com/demdxx/gocast"
)

// 布隆过滤器服务
type BloomService struct {
	m, k      int32
	encryptor Encryptor
	client    *RedisClient
}

// 优化后的构造函数，添加参数验证
func NewBloomService(m, k int32, client *RedisClient, encryptor Encryptor) (*BloomService, error) {
	if m <= 0 || k <= 0 {
		return nil, fmt.Errorf("m and k must be positive, got m=%d, k=%d", m, k)
	}
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor cannot be nil")
	}

	// 初始化Lua脚本
	if err := LoadScripts(); err != nil {
		return nil, fmt.Errorf("failed to load lua scripts: %w", err)
	}

	return &BloomService{
		m:         m,
		k:         k,
		client:    client,
		encryptor: encryptor,
	}, nil
}

// 检查元素是否存在
func (b *BloomService) Exist(ctx context.Context, key, val string) (bool, error) {
	if key == "" || val == "" {
		return false, fmt.Errorf("key and value cannot be empty")
	}

	offsets := b.getKEncrypted(val)
	args := make([]interface{}, 0, len(offsets)+1)
	args = append(args, b.k)

	for _, offset := range offsets {
		args = append(args, offset)
	}

	resp, err := b.client.RunScript(
		ctx,
		bloomGetScript,
		[]string{key}, // KEY列表
		args...,       // ARGV参数
	)
	if err != nil {
		return false, fmt.Errorf("lua script failed: %w", err)
	}

	return gocast.ToInt(resp) == 1, nil
}

// 添加元素到布隆过滤器
func (b *BloomService) Set(ctx context.Context, key, val string) error {
	if key == "" || val == "" {
		return fmt.Errorf("key and value cannot be empty")
	}

	offsets := b.getKEncrypted(val)
	args := make([]interface{}, 0, len(offsets)+1)
	args = append(args, b.k)

	for _, offset := range offsets {
		args = append(args, offset)
	}

	_, err := b.client.RunScript(
		ctx,
		bloomSetScript,
		[]string{key}, // KEY列表
		args...,       // ARGV参数
	)

	if err != nil {
		return fmt.Errorf("lua script failed: %w", err)
	}
	return nil
}

// 获取K个哈希位置（修复负数问题）
func (b *BloomService) getKEncrypted(val string) []int32 {
	encrypteds := make([]int32, 0, b.k)
	origin := val
	for i := 0; int32(i) < b.k; i++ {
		encrypted := b.encryptor.Encrypt(origin)
		// 使用模运算确保offset在[0, m-1]范围内
		offset := (encrypted%b.m + b.m) % b.m
		encrypteds = append(encrypteds, offset)
		origin = gocast.ToString(encrypted)
	}
	return encrypteds
}

// BloomStats 布隆过滤器统计信息
type BloomStats struct {
	TotalBits         int32   `json:"total_bits"`
	HashFunctions     int32   `json:"hash_functions"`
	SetBits           int64   `json:"set_bits"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

// GetStats 获取布隆过滤器统计信息
func (b *BloomService) GetStats(ctx context.Context, key string) (map[string]interface{}, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	// 获取已设置的位数 - 使用int64避免溢出
	cardinality, err := b.client.client.BitCount(ctx, key, nil).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get bit count: %w", err)
	}

	// 计算假阳性率 - 使用int64计算避免溢出
	falsePositiveRate := b.calculateFalsePositiveRate(cardinality)

	return map[string]interface{}{
		"total_bits":          b.m,
		"hash_functions":      b.k,
		"set_bits":            cardinality, // int64类型，避免溢出
		"false_positive_rate": falsePositiveRate,
	}, nil
}

// GetStatsStruct 获取类型安全的统计信息
func (b *BloomService) GetStatsStruct(ctx context.Context, key string) (*BloomStats, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	cardinality, err := b.client.client.BitCount(ctx, key, nil).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get bit count: %w", err)
	}

	falsePositiveRate := b.calculateFalsePositiveRate(cardinality)

	return &BloomStats{
		TotalBits:         b.m,
		HashFunctions:     b.k,
		SetBits:           cardinality,
		FalsePositiveRate: falsePositiveRate,
	}, nil
}

// 计算假阳性率 - 使用int64避免溢出
func (b *BloomService) calculateFalsePositiveRate(setBits int64) float64 {
	if setBits == 0 {
		return 0.0
	}

	// 假阳性率 = (1 - e^(-k*n/m))^k
	// 其中 n 是已插入的元素数量（估算）
	estimatedElements := float64(setBits) / float64(b.k)
	exponent := -float64(b.k) * estimatedElements / float64(b.m)
	return math.Pow(1-math.Exp(exponent), float64(b.k))
}

// BatchSet 批量添加元素到布隆过滤器（使用pipeline）
func (b *BloomService) BatchSet(ctx context.Context, key string, values []string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if len(values) == 0 {
		return nil
	}

	// 使用pipeline批量执行
	pipe := b.client.client.Pipeline()

	for _, val := range values {
		if val == "" {
			continue // 跳过空值
		}

		offsets := b.getKEncrypted(val)
		args := make([]interface{}, 0, len(offsets)+1)
		args = append(args, b.k)
		for _, offset := range offsets {
			args = append(args, offset)
		}

		// 添加到pipeline
		_ = pipe.EvalSha(ctx, bloomSetScript.Hash(), []string{key}, args...)
	}

	// 执行pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("batch set failed: %w", err)
	}

	return nil
}

// 获取布隆过滤器的配置参数
func (b *BloomService) GetConfig() map[string]int32 {
	return map[string]int32{
		"m": b.m,
		"k": b.k,
	}
}

// Clear 清除布隆过滤器
func (b *BloomService) Clear(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// 删除Redis中的key来清除布隆过滤器
	return b.client.client.Del(ctx, key).Err()
}

// Rename 重命名布隆过滤器（原子操作）
func (b *BloomService) Rename(ctx context.Context, oldKey, newKey string) error {
	if oldKey == "" || newKey == "" {
		return fmt.Errorf("old key and new key cannot be empty")
	}

	// 使用Redis的RENAME命令原子性地重命名
	return b.client.client.Rename(ctx, oldKey, newKey).Err()
}
