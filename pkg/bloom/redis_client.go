package bloom

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client redis.Cmdable // 使用Cmdable接口增强兼容性
}

func NewRedisClient(client redis.Cmdable) *RedisClient {
	return &RedisClient{client: client}
}

// 执行预加载的Lua脚本
func (c *RedisClient) RunScript(
	ctx context.Context,
	script *redis.Script,
	keys []string,
	args ...interface{},
) (interface{}, error) {
	return script.Run(ctx, c.client, keys, args...).Result()
}
