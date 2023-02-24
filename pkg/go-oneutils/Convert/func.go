package Convert

import (
	"unsafe"
)

const hextable = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// S2B string转[]byte
func S2B(s string) (b []byte) {
	*(*string)(unsafe.Pointer(&b)) = s
	*(*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&b)) + 2*unsafe.Sizeof(&b))) = len(s)
	return
}

// B2S []byte转string
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
