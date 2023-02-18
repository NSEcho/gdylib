package gdylib

import (
	"bytes"
	"encoding/binary"
	"github.com/lateralusd/gdylib/internal/macho"
	"unsafe"
)

type wrapperHeader struct {
	magic macho.Magic
	m32   macho.Header32
	m64   macho.Header64
}

func configFromOpts(dylib, bin string, opts ...Option) *config {
	c := &config{
		dylibPath: dylib,
		loadType:  DYLIB,
		bts:       new(bytes.Buffer),
	}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func padPath(name string, padding int) []byte {
	div := len(name) / padding
	count := (div+1)*padding - len(name)
	padd := make([]byte, count)

	res := []byte(name)
	res = append(res, padd...)

	return res
}

func stripNull(data []byte) []byte {
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			return data[:i]
		}
	}
	return data
}

func zeroSlice(size int) []byte {
	s := make([]byte, size)
	for i := 0; i < size; i++ {
		s[i] = 0
	}
	return s
}

func (c *config) getCmdSize() uint32 {
	dylibPathSize := len(padPath(c.dylibPath, 8))
	cmdSize := uint32(unsafe.Sizeof(macho.LoadHeader{}))
	cmdSize += uint32(dylibPathSize)
	switch c.loadType {
	case RPATH:
		cmdSize += uint32(unsafe.Sizeof(macho.Rpath{}))
	default:
		cmdSize += uint32(unsafe.Sizeof(macho.Dylib{}))
	}

	rem := cmdSize % 8
	if rem != 0 {
		cmdSize += rem
	}

	return cmdSize
}

func (c *config) writeLoad(lHeader macho.LoadHeader) {
	ct := uint32(unsafe.Sizeof(lHeader))
	binary.Write(c.bts, c.byteOrder, lHeader)
	switch c.loadType {
	case RPATH:
		sz := uint32(unsafe.Sizeof(lHeader))
		sz += uint32(unsafe.Sizeof(macho.Rpath{}))
		rpath := macho.Rpath{sz}
		binary.Write(c.bts, c.byteOrder, rpath)
		ct += uint32(unsafe.Sizeof(rpath))
	default:
		sz := uint32(unsafe.Sizeof(lHeader))
		sz += uint32(unsafe.Sizeof(macho.Dylib{}))
		dylib := macho.Dylib{sz, 0, 0, 0}
		binary.Write(c.bts, c.byteOrder, dylib)
		ct += uint32(unsafe.Sizeof(dylib))
	}
	paddedPath := padPath(c.dylibPath, 8)
	binary.Write(c.bts, binary.LittleEndian, paddedPath)
	ct += uint32(unsafe.Sizeof(paddedPath))

	if ct != lHeader.Size {
		diff := lHeader.Size - ct
		zero := zeroSlice(int(diff))
		c.bts.Write(zero)
	}
}

func (c *config) getByteOrder() {
	c.byteOrder = binary.LittleEndian
}
