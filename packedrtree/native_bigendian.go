//go:build armbe || arm64be || mips || mips64 || mips64p32 || ppc || ppc64 || sparc || sparc64 || s390 || s390x
// +build armbe arm64be mips mips64 mips64p32 ppc ppc64 sparc sparc64 s390 s390x

package packedrtree

func fixLittleEndianOctets(b []byte) {
	for i := 0; i < len(b); i += 8 {
		b[i+0], b[i+7] = b[i+7], b[i+0]
		b[i+1], b[i+6] = b[i+6], b[i+1]
		b[i+2], b[i+5] = b[i+5], b[i+2]
		b[i+3], b[i+4] = b[i+4], b[i+3]
	}
}

func writeLittleEndianOctets(w io.Writer, p []byte) (n int, err error) {
	if len(p)%8 != 0 {
		textPanic("len(p) must be exact multiple of 8")
	}
	buf := make([]byte, 8096)
	for n < len(p) {
		if len(p)-n < len(buf) {
			buf = buf[0 : len(p)-n]
		}
		for i := 0; i < len(buf); i += 8 {
			buf[i+0] = p[n+i+7]
			buf[i+1] = p[n+i+6]
			buf[i+2] = p[n+i+5]
			buf[i+3] = p[n+i+4]
			buf[i+4] = p[n+i+3]
			buf[i+5] = p[n+i+2]
			buf[i+6] = p[n+i+1]
			buf[i+7] = p[n+i+0]
		}
		m, err = w.Write(buf)
		n += m
		if err != nil {
			return
		}
	}
	return
}
