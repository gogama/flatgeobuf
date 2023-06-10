package littleendian

func Uint32(b []byte) uint32 {
	_ = b[3] // Bounds check hint to compiler: see golang.org/issue/14808
	return uint32(b[0]) | uint32(b[1]<<8) | uint32(b[2])<<16 | uint32(b[3])<<24
}
