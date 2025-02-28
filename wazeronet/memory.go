package wazeronet

import (
	"unsafe"

	"github.com/egdaemon/wasinet/wasinet/ffi"
)

type memory interface {
	// ReadUint32Le reads a uint32 in little-endian encoding from the underlying buffer at the offset in or returns
	// false if out of range.
	ReadUint32Le(offset uint32) (uint32, bool)

	Read(offset, byteCount uint32) ([]byte, bool)
	// WriteUint32Le writes the value in little-endian encoding to the underlying buffer at the offset in or returns
	// false if out of range.
	WriteUint32Le(offset, v uint32) bool
	// Write writes the slice to the underlying buffer at the offset or returns false if out of range.
	Write(offset uint32, v []byte) bool
}

func Memory(m memory) ffi.Memory {
	return memoryadapter{m: m}
}

type memoryadapter struct {
	m memory
}

func (t memoryadapter) Read(offset unsafe.Pointer, byteCount uint32) ([]byte, bool) {
	return t.m.Read(uint32(uintptr(offset)), byteCount)
}

func (t memoryadapter) Write(offset unsafe.Pointer, v []byte) bool {
	return t.m.Write(uint32(uintptr(offset)), v)
}

func (t memoryadapter) ReadUint32Le(offset unsafe.Pointer) (uint32, bool) {
	return t.m.ReadUint32Le(uint32(uintptr(offset)))
}

func (t memoryadapter) WriteUint32Le(offset unsafe.Pointer, v uint32) bool {
	return t.m.WriteUint32Le(uint32(uintptr(offset)), v)
}
