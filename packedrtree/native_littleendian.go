//go:build 386 || amd64 || amd64p32 || arm || arm64 || loong64 || mipsle || mis64le || mips64p32le || ppc64le || riscv || riscv64 || wasm
// +build 386 amd64 amd64p32 arm arm64 loong64 mipsle mis64le mips64p32le ppc64le riscv riscv64 wasm

package packedrtree

func fixLittleEndianOctets(_ []byte) {} // No-op since architecture is little-endian.
