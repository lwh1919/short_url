package generator

import "testing"

func TestShortUrlService_generateShortUrl(t *testing.T) {
	testCases := []struct {
		name      string
		originUrl string
		suffix    string
		weights   []int
	}{
		{
			name:      "正常生成1",
			originUrl: "https://example.com",
			suffix:    "",
			weights:   []int{5, 67, 23, 71, 73, 79},
		},
		{
			name:      "正常生成2",
			originUrl: "http://123456789.com/1563sdf/set",
			suffix:    "",
			weights:   []int{5, 67, 23, 71, 73, 79},
		},
		{
			name:      "模拟生成冲突1",
			originUrl: "https://example.com",
			suffix:    "SHORTURL",
			weights:   []int{5, 67, 23, 71, 73, 79},
		},
		{
			name:      "模拟生成冲突2",
			originUrl: "http://123456789.com/1563sdf/set",
			suffix:    "SHORTURL",
			weights:   []int{5, 67, 23, 71, 73, 79},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateShortUrl(tc.originUrl, tc.suffix, tc.weights)
			t.Log(got)
		})
	}
}
