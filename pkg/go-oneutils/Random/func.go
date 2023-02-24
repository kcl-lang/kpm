package Random

const hextable = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const hextable64 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01"

// RandBytes 快速生成n位随机字节
func RandBytes(n int) []byte {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = hextable64[RandInt64()]
	}
	return b
}

// RandBytes32 快速生成32位随机字节
func RandBytes32() []byte {
	b := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b[i] = hextable64[RandInt64()]
	}
	return b
}

// RandIntn  快速生成num范围内的随机数
func RandIntn(num uint32) uint32 {
	return FastRand() % num
}

// RandInt64 快速生成64内的随机数 0-63
func RandInt64() uint32 {
	i := FastRand() & 63
	return i
}
func RandInt36() uint32 {
	return FastRand() & 35
}

// UUIDv4 快速生成UUIDv4
func UUIDv4() []byte {
	b := make([]byte, 36)
	for i := 0; i < 30; i++ {
		b[i] = hextable[FastRand()&35]
	}
	b[30], b[31], b[32], b[33], b[34], b[35],
		b[8], b[13], b[14], b[18], b[19], b[23] = b[8], b[13], b[14], b[18], b[19], b[23],
		'-', '-', '4', '-', 'a', '-'
	return b
}
