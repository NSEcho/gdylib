package gdylib

import (
	"bytes"
	"encoding/binary"
	"github.com/lateralusd/gdylib/internal/macho"
	"io"
	"os"
	"unsafe"
)

type LoadType uint32

const (
	WEAK LoadType = iota
	DYLIB
	RPATH
)

type config struct {
	loadType   LoadType
	dylibPath  string
	binaryPath string
	bts        *bytes.Buffer
	f          *os.File
}

type Option = func(c *config)

func Run(binaryPath, dylibPath string, opts ...Option) (io.Reader, error) {
	c := &config{
		dylibPath: dylibPath,
		loadType:  DYLIB,
		bts:       new(bytes.Buffer),
	}
	for _, opt := range opts {
		opt(c)
	}

	f, err := os.OpenFile(binaryPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c.f = f

	var hdr macho.Header64
	binary.Read(c.f, binary.LittleEndian, &hdr)

	if hdr.Filetype != macho.MH_EXECUTE {
		return nil, ErrNotExecute
	}

	var loads []macho.Load
	var lcDataSize uint32

	off, _ := c.f.Seek(0, io.SeekCurrent)
	for i := 0; i < int(hdr.NCmds); i++ {
		var ld macho.LoadHeader
		binary.Read(c.f, binary.LittleEndian, &ld)
		switch ld.Cmd {
		case macho.LC_CODE_SIGNATURE:
			if i != int(hdr.NCmds-1) {
				return nil, ErrNotLastCommand
			}

			var lcCode macho.LCCode
			binary.Read(c.f, binary.LittleEndian, &lcCode)
			lcDataSize = lcCode.DataSize

			f.Seek(off, 0)
			// write zero in the place where LC_CODE_SIGNATURE was
			buffer := macho.ZeroSlice(int(ld.Size))
			loads = append(loads, macho.Load{
				LoadHeader: macho.LoadHeader{Cmd: 0, Size: 0},
				Raw:        buffer,
			})
		default:
			f.Seek(off, 0)
			buffer := make([]byte, ld.Size)
			binary.Read(c.f, binary.LittleEndian, &buffer)
			loads = append(loads, macho.Load{
				LoadHeader: macho.LoadHeader{Cmd: ld.Cmd, Size: ld.Size},
				Raw:        buffer,
			})
		}
		off += int64(ld.Size)
		f.Seek(off, 0)
	}

	dylibPathSize := len(padPath(c.dylibPath, 8))

	cmdSize := int(unsafe.Sizeof(macho.LoadHeader{})) + int(unsafe.Sizeof(macho.Dylib{})) + dylibPathSize

	var loadHeader macho.LoadHeader
	if c.loadType == DYLIB {
		loadHeader.Cmd = macho.LC_LOAD_DYLIB
	} else {
		loadHeader.Cmd = macho.LC_LOAD_WEAK_DYLIB
	}
	loadHeader.Size = uint32(cmdSize)

	hdr.SizeOfCmds -= 16
	hdr.SizeOfCmds += uint32(cmdSize)

	binary.Write(c.bts, binary.LittleEndian, hdr)

	end, _ := c.f.Seek(0, io.SeekEnd)

	for _, load := range loads {
		found := true
		for _, b := range load.Raw {
			if b != 0 {
				found = false
				break
			}
		}
		if !found {
			switch load.Cmd {
			case macho.LC_SEGMENT_64:
				var seg macho.Segment64
				bt := bytes.NewBuffer(load.Raw[8:])
				binary.Read(bt, binary.LittleEndian, &seg)
				if string(stripNull(seg.SegName[:])) == "__LINKEDIT" {
					seg.FileSize -= uint64(lcDataSize)
					newL := new(bytes.Buffer)
					binary.Write(newL, binary.LittleEndian, load.LoadHeader)
					binary.Write(newL, binary.LittleEndian, seg)
					load.Raw = newL.Bytes()
				}
			case macho.LC_SYMTAB:
				var symtab macho.Symtab
				bt := bytes.NewBuffer(load.Raw[8:])
				binary.Read(bt, binary.LittleEndian, &symtab)
				sz := end - int64(lcDataSize)
				diffSize := int64(symtab.StrOff+symtab.StrSize) - sz
				newSize := symtab.StrSize - uint32(diffSize)
				symtab.StrSize = newSize
				newS := new(bytes.Buffer)
				binary.Write(newS, binary.LittleEndian, load.LoadHeader)
				binary.Write(newS, binary.LittleEndian, symtab)
				load.Raw = newS.Bytes()
			}
			c.bts.Write(load.Raw)
		}
	}

	// write lc_load_dylib header
	binary.Write(c.bts, binary.LittleEndian, loadHeader)

	// write dylib struct
	dlib := macho.Dylib{24, 0, 0, 0}
	binary.Write(c.bts, binary.LittleEndian, dlib)

	// calculate padded path and write it to new file
	paddedPath := padPath(c.dylibPath, 8)
	binary.Write(c.bts, binary.LittleEndian, paddedPath)

	currentOff := int64(unsafe.Sizeof(hdr)) + int64(hdr.SizeOfCmds)

	count := end - int64(lcDataSize) - currentOff
	rest := make([]byte, count)
	f.Seek(currentOff, io.SeekStart)
	binary.Read(c.f, binary.LittleEndian, &rest)

	c.bts.Write(rest)

	return c.bts, nil
}

func WithLoadType(t LoadType) Option {
	return Option(func(c *config) {
		c.loadType = t
	})
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
