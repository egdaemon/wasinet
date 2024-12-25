package ffi_test

import (
	"testing"

	"github.com/egdaemon/wasinet/ffi"
	"github.com/stretchr/testify/require"
)

func TestRawWrite(t *testing.T) {
	type m struct {
		Foo int32
	}
	var (
		exv  = m{}
		exv2 = &m{}
	)

	exvptr, exvlength := ffi.Pointer(&exv)
	err := ffi.RawWrite(ffi.Native{}, &m{Foo: 512}, exvptr, exvlength)
	require.NoError(t, err)
	require.Equal(t, int32(512), exv.Foo)
	exv2ptr, _ := ffi.Pointer(exv2)
	err = ffi.RawRead(ffi.Native{}, exv2ptr, exvptr, exvlength)
	require.NoError(t, err)
	require.Equal(t, int32(512), exv2.Foo)
}

func TestUint32(t *testing.T) {
	var (
		ex uint32
	)
	exptr, exlen := ffi.Pointer(&ex)
	err := ffi.Uint32Write(ffi.Native{}, exptr, 42)
	require.NoError(t, err)
	exv, err := ffi.Uint32Read(ffi.Native{}, exptr, exlen)
	require.NoError(t, err)
	require.Equal(t, uint32(42), exv)
}
