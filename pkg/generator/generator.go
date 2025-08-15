package generator

import (
	"crypto/sha256"
)

func GenerateShortUrl(originUrl, suffix string, weights []int) string {
	hash := sha256.Sum256([]byte(originUrl + suffix))

	// 计算前导零和有效字节
	leadingZeros := countLeadingZeros(hash[:])
	encoded := encodeBase62(hash[:])

	// 确保至少6个有效字符
	if len(encoded) < 6 {
		encoded = append(encoded, make([]byte, 6-len(encoded))...)
	}

	// 构建基础短链
	shortPart := encoded[:6]
	checksum := calculateChecksum(shortPart, weights)

	// 插入校验位到第4个位置
	result := make([]byte, 7)
	copy(result[:3], shortPart[:3])
	result[3] = BASE62CHARSET[checksum]
	copy(result[4:], shortPart[3:6])

	// 添加前导零
	return string(append(make([]byte, leadingZeros), result...))
}

func CheckShortUrl(shortUrl string, weights []int) bool {
	if len(shortUrl) != 7+len(shortUrl)-len(shortUrl[:7]) { // 保持原始长度校验
		return false
	}

	// 提取校验位
	checksumChar := shortUrl[3]
	data := append([]byte(shortUrl[:3]), shortUrl[4:]...)

	return Base62NumberTable[string(checksumChar)] == calculateChecksum(data[:6], weights)
}

// 优化后的前导零计算
func countLeadingZeros(data []byte) int {
	for i, b := range data {
		if b != 0 {
			return i
		}
	}
	return len(data)
}

// 快速base62编码
func encodeBase62(data []byte) []byte {
	var result []byte
	var buffer uint32

	bitsAvailable := 0
	for _, b := range data {
		buffer = (buffer << 8) | uint32(b)
		bitsAvailable += 8

		for bitsAvailable >= 6 {
			bitsAvailable -= 6
			index := (buffer >> bitsAvailable) & 0x3F
			index %= 62 // 确保索引在0-61范围内
			result = append(result, BASE62CHARSET[index])
		}
	}

	if bitsAvailable > 0 {
		index := (buffer << (6 - bitsAvailable)) & 0x3F
		index %= 62 // 确保索引在0-61范围内
		result = append(result, BASE62CHARSET[index])
	}

	return result
}

// 优化校验和计算
func calculateChecksum(data []byte, weights []int) int {
	sum := 0
	for i, c := range data[:6] { // 确保只处理前6个字符
		sum += Base62NumberTable[string(c)] * weights[i]
	}
	return sum % 62
}
