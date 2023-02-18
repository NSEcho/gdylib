package gdylib

import (
	"bytes"
	"encoding/binary"
	"github.com/lateralusd/gdylib/internal/macho"
	"io"
	"os"
)

type LoadType uint32

const (
	WEAK LoadType = iota
	DYLIB
	RPATH
)

type config struct {
	loadType      LoadType
	removeCodeSig bool
	dylibPath     string
	binaryPath    string
	byteOrder     binary.ByteOrder
	bts           *bytes.Buffer
	loads         *bytes.Buffer
	f             *os.File
}

type Option = func(c *config)

func Run(binaryPath, dylibPath string, opts ...Option) (io.Reader, error) {
	c := configFromOpts(dylibPath, binaryPath, opts...)

	f, err := os.OpenFile(binaryPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c.f = f
	c.getByteOrder()

	start := make([]byte, 4)
	c.f.ReadAt(start, 0)
	magic := binary.LittleEndian.Uint32(start)

	var wHeader wrapperHeader
	wHeader.magic = macho.Magic(magic)

	if wHeader.magic == macho.M32 {
		binary.Read(c.f, binary.LittleEndian, &wHeader.m32)
	} else {
		binary.Read(c.f, binary.LittleEndian, &wHeader.m64)
	}

	var filetype macho.Filetype
	var ncmds uint32
	var sizeofcmds uint32

	if wHeader.magic == macho.M32 {
		filetype = wHeader.m32.Filetype
		ncmds = wHeader.m32.NCmds
		sizeofcmds = wHeader.m32.SizeOfCmds
	} else {
		filetype = wHeader.m64.Filetype
		ncmds = wHeader.m64.NCmds
		sizeofcmds = wHeader.m64.SizeOfCmds
	}

	if filetype != macho.MH_EXECUTE {
		return nil, ErrNotExecute
	}

	var loads []macho.Load
	var lcDataSize uint32

	off, _ := c.f.Seek(0, io.SeekCurrent)
	for i := 0; i < int(ncmds); i++ {
		var ld macho.LoadHeader
		binary.Read(c.f, binary.LittleEndian, &ld)
		switch ld.Cmd {
		case macho.LC_CODE_SIGNATURE:
			if c.removeCodeSig {
				if i != int(ncmds-1) {
					return nil, ErrNotLastCommand
				}

				var lcCode macho.LCCode
				binary.Read(c.f, binary.LittleEndian, &lcCode)
				lcDataSize = lcCode.DataSize
				f.Seek(off, 0)
				// write zero in the place where LC_CODE_SIGNATURE was
				buffer := zeroSlice(int(ld.Size))
				loads = append(loads, macho.Load{
					LoadHeader: macho.LoadHeader{Cmd: ld.Cmd, Size: ld.Size},
					Raw:        buffer,
				})
			} else {
				f.Seek(off, 0)
				buffer := make([]byte, ld.Size)
				binary.Read(c.f, c.byteOrder, &buffer)
				loads = append(loads, macho.Load{
					LoadHeader: macho.LoadHeader{Cmd: ld.Cmd, Size: ld.Size},
					Raw:        buffer,
				})
			}
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

	var loadHeader macho.LoadHeader
	switch c.loadType {
	case DYLIB:
		loadHeader.Cmd = macho.LC_LOAD_DYLIB
	case WEAK:
		loadHeader.Cmd = macho.LC_LOAD_WEAK_DYLIB
	case RPATH:
		loadHeader.Cmd = macho.LC_RPATH
	default:
		return nil, ErrTypeNotSupported
	}

	cmdSize := c.getCmdSize()

	loadHeader.Size = cmdSize

	cmdBuffer := make([]byte, cmdSize)
	if c.removeCodeSig {
		for _, load := range loads {
			switch load.Cmd {
			case macho.LC_CODE_SIGNATURE:
				for i := 0; i < len(load.Raw); i++ {
					cmdBuffer[i] = load.Raw[i]
				}
				rest := make([]byte, len(cmdBuffer)-len(load.Raw))
				f.ReadAt(rest, off)
				for i := 0; i < len(rest); i++ {
					cmdBuffer[i+len(load.Raw)] = rest[i]
				}
				ncmds -= 1
				sizeofcmds -= load.Size
			}
		}
	} else {
		f.ReadAt(cmdBuffer, off)
	}

	hasSpace := true
	for _, b := range cmdBuffer {
		if b != 0 {
			hasSpace = false
		}
	}

	if !hasSpace {
		return nil, ErrNotEnoughSpace
	}

	if wHeader.magic == macho.M32 {
		wHeader.m32.SizeOfCmds = sizeofcmds
		wHeader.m32.NCmds = ncmds

		wHeader.m32.NCmds += 1
		wHeader.m32.SizeOfCmds += cmdSize

		binary.Write(c.bts, binary.LittleEndian, wHeader.m32)
	} else {
		wHeader.m64.SizeOfCmds = sizeofcmds
		wHeader.m64.NCmds = ncmds

		wHeader.m64.NCmds += 1
		wHeader.m64.SizeOfCmds += cmdSize

		binary.Write(c.bts, binary.LittleEndian, wHeader.m64)
	}

	end, _ := c.f.Seek(0, io.SeekEnd)

	// write the loads to buffer
	for _, load := range loads {
		if c.removeCodeSig {
			if load.Cmd != macho.LC_CODE_SIGNATURE {
				switch load.Cmd {
				case macho.LC_SEGMENT:
					var seg macho.Segment
					bt := bytes.NewBuffer(load.Raw[8:])
					binary.Read(bt, binary.LittleEndian, &seg)
					if string(stripNull(seg.SegName[:])) == "__LINKEDIT" {
						seg.FileSize -= lcDataSize
						newL := new(bytes.Buffer)
						binary.Write(newL, binary.LittleEndian, load.LoadHeader)
						binary.Write(newL, binary.LittleEndian, seg)
						load.Raw = newL.Bytes()
					}
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
		} else {
			c.bts.Write(load.Raw)
		}
	}

	c.writeLoad(loadHeader)

	currentOff := len(c.bts.Bytes())

	count := end - int64(currentOff)
	if c.removeCodeSig {
		count -= int64(lcDataSize)
	}
	rest := make([]byte, count)
	f.Seek(int64(currentOff), io.SeekStart)
	binary.Read(c.f, binary.LittleEndian, &rest)

	c.bts.Write(rest)

	return c.bts, nil
}

func WithLoadType(t LoadType) Option {
	return Option(func(c *config) {
		c.loadType = t
	})
}

func WithRemoveCodeSig(remove bool) Option {
	return Option(func(c *config) {
		c.removeCodeSig = remove
	})
}
