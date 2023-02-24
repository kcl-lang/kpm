package GlobalStore

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"github.com/cespare/xxhash/v2"
)

const hextable = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type Integrity string

// GetType 获取hash类型
func (it Integrity) GetType() string {
	for i := 0; i < len(it); i++ {
		if it[i] == '-' {
			return string(it[:i])
		}
	}
	return ""
}

// GetValue 获取直接来自Integrity的值（hash+base64计算得到）
func (it Integrity) GetValue() string {
	itlen := len(it)
	for i := 0; i < itlen; i++ {
		if it[i] == '-' {
			if itlen > i+1 {
				return string(it[i+1:])
			}
			return ""
		}
	}
	return ""
}

// GetRawHashValue 获取原始hash值
func (it Integrity) GetRawHashValue() ([]byte, error) {
	decodeString, err := base64.StdEncoding.DecodeString(it.GetValue())
	if err != nil {
		return nil, err
	}
	return decodeString, nil
}

// GetRawHashString 获取原始hash字符串值 hash to string
func (it Integrity) GetRawHashString() (string, error) {
	value, err := it.GetRawHashValue()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
func (it Integrity) GetRawHashStringAndMod256() (string, string, error) {
	hashRawValue, err := it.GetRawHashValue()
	if err != nil {
		return "", "", err
	}
	t := xxhash.Sum64(hashRawValue) & 255
	return hex.EncodeToString(hashRawValue), string([]byte{hextable[t>>4], hextable[t&15]}), nil
}
func (it Integrity) GetRawHashStringAndMod(i uint64) (string, string, error) {
	hashRawValue, err := it.GetRawHashValue()
	if err != nil {
		return "", "", err
	}
	t := xxhash.Sum64(hashRawValue) % i
	return hex.EncodeToString(hashRawValue), string([]byte{hextable[t>>4], hextable[t&15]}), nil
}

// GetRawHashStringAndModFast 获取原始hash字符串并且快速取模（i为16的n次方）
func (it Integrity) GetRawHashStringAndModFast(i uint64) (string, string, error) {
	hashRawValue, err := it.GetRawHashValue()
	if err != nil {
		return "", "", err
	}
	t := xxhash.Sum64(hashRawValue) & (i - 1)
	return hex.EncodeToString(hashRawValue), string([]byte{hextable[t>>4], hextable[t&15]}), nil
}

// NewIntegrity 新建Integrity实例
func NewIntegrity(integrityType string, data []byte) Integrity {
	var integrityBytes = make([]byte, 64)
	integrityBytes = integrityBytes[:0]
	switch integrityType {
	case "md5":
		t := md5.Sum(data)
		integrityBytes = append(integrityBytes, t[:]...)
	case "sha1":
		t := sha1.Sum(data)
		integrityBytes = append(integrityBytes, t[:]...)
	case "sha256":
		t := sha256.Sum256(data)
		integrityBytes = append(integrityBytes, t[:]...)
	case "sha512":
		t := sha512.Sum512(data)
		integrityBytes = append(integrityBytes, t[:]...)
	}

	return Integrity(integrityType + "-" + base64.StdEncoding.EncodeToString(integrityBytes))
}
