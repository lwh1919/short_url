package bloom

import (
	"embed"
	"github.com/redis/go-redis/v9"
)

//go:embed scripts/*.lua
var luaScripts embed.FS

// 预加载的Lua脚本
var (
	bloomGetScript *redis.Script
	bloomSetScript *redis.Script
)

func LoadScripts() error {
	getSrc, err := luaScripts.ReadFile("scripts/bloom_get.lua")
	if err != nil {
		return err
	}

	setSrc, err := luaScripts.ReadFile("scripts/bloom_set.lua")
	if err != nil {
		return err
	}

	bloomGetScript = redis.NewScript(string(getSrc))
	bloomSetScript = redis.NewScript(string(setSrc))
	return nil
}
