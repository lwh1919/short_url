package epay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"short_url/pkg/sign"
	"sort"
	"strings"
)

type EpaySignHandler struct {
}

var _ sign.SignHandler = (*EpaySignHandler)(nil)

func NewEpaySignHandler() sign.SignHandler {
	return &EpaySignHandler{}
}

func (h *EpaySignHandler) GenerateSign(data map[string]any, apiKey string) (ret string, err error) {
	// 防止 panic 意外中断程序
	defer func() {
		if r := recover(); r != nil {
			ret = ""
			err = fmt.Errorf("%v", r)
		}
	}()

	if apiKey == "" {
		panic("apiKey cannot be empty")
	}

	// Step 1: 过滤空值
	cleaned := h.removeEmptyValues(data)

	// Step 2: 生成排序后的JSON
	sortedJSON, err := h.sortedJSONString(cleaned)
	if err != nil {
		return "", err
	}

	// Step 3: 转换QueryString
	var raw map[string]any
	if err := json.Unmarshal([]byte(sortedJSON), &raw); err != nil {
		return "", err
	}
	queryStr := h.toQueryString(raw)

	queryStr = strings.ReplaceAll(queryStr, "}&{", "&")

	// Step 4: 拼接API密钥
	finalStr := fmt.Sprintf("%s&key=%s", queryStr, apiKey)

	// Step 5: SHA256签名
	hash := sha256.Sum256([]byte(finalStr))
	return strings.ToUpper(hex.EncodeToString(hash[:])), nil
}

func (h *EpaySignHandler) removeEmptyValues(data map[string]any) map[string]any {
	cleaned := make(map[string]any)
	for k, v := range data {
		switch val := v.(type) {
		case nil:
			continue // 跳过nil值
		case string:
			if val != "" {
				cleaned[k] = val
			}
		case map[string]any:
			if nested := h.removeEmptyValues(val); len(nested) > 0 {
				cleaned[k] = nested
			}
		default:
			cleaned[k] = val // 保留非空基础类型
		}
	}
	return cleaned
}

func (h *EpaySignHandler) sortedJSONString(data map[string]any) (string, error) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Go默认按ASCII排序

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, "%q:", k)

		switch v := data[k].(type) {
		case map[string]any:
			nested, _ := h.sortedJSONString(v)
			buf.WriteString(nested)
		default:
			enc := json.NewEncoder(&buf)
			enc.SetEscapeHTML(false)
			enc.Encode(v)
			buf.Truncate(buf.Len() - 1) // 移除换行符
		}
	}
	buf.WriteByte('}')
	return buf.String(), nil
}

func (h *EpaySignHandler) toQueryString(data map[string]any) string {
	var builder strings.Builder
	h.encodeValue(&builder, data, true)
	return strings.TrimSuffix(builder.String(), "&")
}

func (h *EpaySignHandler) encodeValue(b *strings.Builder, data map[string]any, topLevel bool) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		if !topLevel {
			b.WriteString("{")
		}

		switch val := v.(type) {
		case map[string]any:
			b.WriteString(k + "=")
			h.encodeValue(b, val, false)
		default:
			fmt.Fprintf(b, "%s=%v&", k, val)
		}

		if !topLevel {
			str := b.String()
			b.Reset()
			b.WriteString(strings.TrimSuffix(str, "&") + "}&")
		}
	}
}
