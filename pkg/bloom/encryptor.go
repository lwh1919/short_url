package bloom

import (
	"crypto/md5"
	"encoding/binary"
	"hash/fnv"
)

// Encryptor 接口定义哈希加密方法
type Encryptor interface {
	Encrypt(val string) int32
}

// DefaultEncryptor 默认的加密器实现
type DefaultEncryptor struct{}

// NewDefaultEncryptor 创建默认加密器
func NewDefaultEncryptor() *DefaultEncryptor {
	return &DefaultEncryptor{}
}

// Encrypt 使用FNV-1a哈希算法加密字符串
func (e *DefaultEncryptor) Encrypt(val string) int32 {
	h := fnv.New32a()
	h.Write([]byte(val))
	return int32(h.Sum32())
}

// MD5Encryptor MD5加密器实现
type MD5Encryptor struct{}

// NewMD5Encryptor 创建MD5加密器
func NewMD5Encryptor() *MD5Encryptor {
	return &MD5Encryptor{}
}

// Encrypt 使用MD5哈希算法加密字符串
func (e *MD5Encryptor) Encrypt(val string) int32 {
	h := md5.New()
	h.Write([]byte(val))
	hash := h.Sum(nil)
	// 取前4字节转换为int32
	return int32(binary.BigEndian.Uint32(hash[:4]))
}
