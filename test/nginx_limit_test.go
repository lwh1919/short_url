package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNginxLimiter(t *testing.T) {
	const (
		url        = "http://localhost:8888/api/create"
		requests   = 100
		concurrent = 5
	)

	// 生成随机字符串
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ+=")
	generateRandom := func(n int) string {
		b := make([]rune, n)
		for i := range b {
			b[i] = chars[rand.Intn(len(chars))]
		}
		return string(b)
	}

	// 检查服务
	if _, err := http.Get("http://localhost:8888/health"); err != nil {
		t.Skip("nginx服务未启动")
	}

	// 准备请求
	body := map[string]string{
		"origin_url": "https://example.com/" + generateRandom(180),
	}
	jsonBody, _ := json.Marshal(body)

	// 并发测试
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrent)
	stats := struct {
		mu       sync.Mutex
		success  int
		limited  int
		failures int
	}{}

	fmt.Println("🚀 测试Nginx限流 (8888)...")
	start := time.Now()

	for i := 0; i < requests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(id int) {
			defer wg.Done()
			defer func() { <-sem }()

			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				stats.mu.Lock()
				stats.failures++
				stats.mu.Unlock()
				return
			}
			defer resp.Body.Close()

			stats.mu.Lock()
			switch resp.StatusCode {
			case 200:
				stats.success++
				fmt.Printf("✅ %d: 成功\n", id)
			case 429, 503:
				stats.limited++
				fmt.Printf("⚠️  %d: 被nginx限流\n", id)
			default:
				stats.failures++
				fmt.Printf("❌ %d: 错误 %d\n", id, resp.StatusCode)
			}
			stats.mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("📊 结果: 成功=%d, 限流=%d, 失败=%d, 耗时=%v\n",
		stats.success, stats.limited, stats.failures, time.Since(start))
}
