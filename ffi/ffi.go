package ffi

import (
	"context"
	"errors"
	"time"
)

type memory interface {

	// // ReadUint32Le reads a uint32 in little-endian encoding from the underlying buffer at the offset in or returns
	// // false if out of range.
	ReadUint32Le(offset uint32) (uint32, bool)

	// // ReadFloat32Le reads a float32 from 32 IEEE 754 little-endian encoded bits in the underlying buffer at the offset
	// // or returns false if out of range.
	// // See math.Float32bits
	// ReadFloat32Le(offset uint32) (float32, bool)

	// // ReadUint64Le reads a uint64 in little-endian encoding from the underlying buffer at the offset or returns false
	// // if out of range.
	// ReadUint64Le(offset uint32) (uint64, bool)

	// // ReadFloat64Le reads a float64 from 64 IEEE 754 little-endian encoded bits in the underlying buffer at the offset
	// // or returns false if out of range.
	// //
	// // See math.Float64bits
	// ReadFloat64Le(offset uint32) (float64, bool)

	Read(offset, byteCount uint32) ([]byte, bool)

	// // WriteByte writes a single byte to the underlying buffer at the offset in or returns false if out of range.
	// WriteByte(offset uint32, v byte) bool

	// // WriteUint16Le writes the value in little-endian encoding to the underlying buffer at the offset in or returns
	// // false if out of range.
	// WriteUint16Le(offset uint32, v uint16) bool

	// // WriteUint32Le writes the value in little-endian encoding to the underlying buffer at the offset in or returns
	// // false if out of range.
	// WriteUint32Le(offset, v uint32) bool

	// // WriteFloat32Le writes the value in 32 IEEE 754 little-endian encoded bits to the underlying buffer at the offset
	// // or returns false if out of range.
	// //
	// // See math.Float32bits
	// WriteFloat32Le(offset uint32, v float32) bool

	// // WriteUint64Le writes the value in little-endian encoding to the underlying buffer at the offset in or returns
	// // false if out of range.
	// WriteUint64Le(offset uint32, v uint64) bool

	// // WriteFloat64Le writes the value in 64 IEEE 754 little-endian encoded bits to the underlying buffer at the offset
	// // or returns false if out of range.
	// //
	// // See math.Float64bits
	// WriteFloat64Le(offset uint32, v float64) bool

	// // Write writes the slice to the underlying buffer at the offset or returns false if out of range.
	Write(offset uint32, v []byte) bool

	// // Definition is metadata about this memory from its defining module.
	// Definition() MemoryDefinition

	// // Size returns the memory size in bytes available.
	// // e.g. If the underlying memory has 1 page: 65536
	// //
	// // # Notes
	// //
	// //   - This overflows (returns zero) if the memory has the maximum 65536 pages.
	// // 	   As a workaround until wazero v2 to fix the return type, use Grow(0) to obtain the current pages and
	// //     multiply by 65536.
	// //
	// // See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#-hrefsyntax-instr-memorymathsfmemorysize%E2%91%A0
	// Size() uint32

	// // Grow increases memory by the delta in pages (65536 bytes per page).
	// // The return val is the previous memory size in pages, or false if the
	// // delta was ignored as it exceeds MemoryDefinition.Max.
	// //
	// // # Notes
	// //
	// //   - This is the same as the "memory.grow" instruction defined in the
	// //	   WebAssembly Core Specification, except returns false instead of -1.
	// //   - When this returns true, any shared views via Read must be refreshed.
	// //
	// // See MemorySizer Read and https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#grow-mem
	// Grow(deltaPages uint32) (previousPages uint32, ok bool)

	// // ReadByte reads a single byte from the underlying buffer at the offset or returns false if out of range.
	// ReadByte(offset uint32) (byte, bool)

	// // ReadUint16Le reads a uint16 in little-endian encoding from the underlying buffer at the offset in or returns
	// // false if out of range.
	// ReadUint16Le(offset uint32) (uint16, bool)	// // WriteString writes the string to the underlying buffer at the offset or returns false if out of range.
	// WriteString(offset uint32, v string) bool
}

func ReadString(m memory, offset uint32, len uint32) (string, error) {
	var (
		ok   bool
		data []byte
	)

	if data, ok = m.Read(offset, len); !ok {
		return "", errors.New("unable to read string")
	}

	return string(data), nil
}

func ReadStringArray(m memory, offset uint32, length uint32, argssize uint32) (args []string, err error) {
	args = make([]string, 0, length)

	for offset, i := offset, uint32(0); i < length*2; offset, i = offset+(2*argssize), i+2 {
		var (
			data []byte
		)

		if data, err = ReadArrayElement(m, offset, argssize); err != nil {
			return nil, err
		}

		args = append(args, string(data))
	}

	return args, nil
}

func ReadArrayElement(m memory, offset, len uint32) (data []byte, err error) {
	var (
		ok            bool
		eoffset, elen uint32
	)

	if eoffset, ok = m.ReadUint32Le(offset); !ok {
		return nil, errors.New("unable to read element offset")
	}

	if elen, ok = m.ReadUint32Le(offset + len); !ok {
		return nil, errors.New("unable to read element byte length")
	}

	if data, ok = m.Read(eoffset, elen); !ok {
		return nil, errors.New("unable to read element bytes")
	}

	return data, nil
}

func ReadMicroDeadline(ctx context.Context, deadline int64) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, time.UnixMicro(deadline))
}

func ReadBytes(m memory, offset uint32, len uint32) (data []byte, err error) {
	var (
		ok bool
	)

	if data, ok = m.Read(offset, len); !ok {
		return nil, errors.New("unable to read string")
	}

	return data, nil
}
