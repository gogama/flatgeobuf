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
