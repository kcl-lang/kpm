package Random

import _ "unsafe"

//go:linkname FastRand runtime.fastrand
func FastRand() uint32
