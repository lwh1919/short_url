package generator

import (
	"crypto/sha256"
)

// GenerateShortUrl 生成固定长度为7的短链接
// 修复了前导零问题，确保不会生成包含空字符的链接
func GenerateShortUrl(originUrl, suffix string, weights []int) string {
	hash := sha256.Sum256([]byte(originUrl + suffix))

	// 使用哈希值生成Base62编码
	encoded := encodeBase62(hash[:])

	// 确保至少有6个有效字符，如果不够则使用BASE62CHARSET[0]填充
	for len(encoded) < 6 {
		encoded = append(encoded, BASE62CHARSET[0])
	}

	// 取前6个字符作为基础短链
	shortPart := encoded[:6]

	// 计算校验和
	checksum := calculateChecksum(shortPart, weights)

	// 插入校验位到第4个位置，形成7位短链接
	result := make([]byte, 7)
	copy(result[:3], shortPart[:3])
	result[3] = BASE62CHARSET[checksum]
	copy(result[4:], shortPart[3:6])

	return string(result)
}

// CheckShortUrl 验证短链接的有效性
func CheckShortUrl(shortUrl string, weights []int) bool {
	// 确保短链接长度为7
	if len(shortUrl) != 7 {
		return false
	}

	// 提取校验位
	checksumChar := shortUrl[3]

	// 构建用于校验的数据
	data := append([]byte(shortUrl[:3]), shortUrl[4:]...)

	// 验证校验和
	return Base62NumberTable[string(checksumChar)] == calculateChecksum(data, weights)
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
		charValue, ok := Base62NumberTable[string(c)]
		if !ok {
			// 如果字符不在Base62表中，使用默认值0
			charValue = 0
		}
		sum += charValue * weights[i]
	}
	return sum % 62
}
