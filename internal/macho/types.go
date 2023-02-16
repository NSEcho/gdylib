package macho

type Magic uint32

const (
	M32  Magic = 0xfeedface
	M64  Magic = 0xfeedfacf
	MFat Magic = 0xcafebabe
)

type Filetype uint32

const (
	MH_EXECUTE Filetype = 0x2
)

type Cmd uint32

const (
	LC_REQ_DYLD        Cmd = 0x80000000
	LC_SEGMENT         Cmd = 0x1
	LC_SYMTAB          Cmd = 0x2
	LC_LOAD_DYLIB      Cmd = 0xc
	LC_LOAD_WEAK_DYLIB     = (0x18 | LC_REQ_DYLD)
	LC_SEGMENT_64      Cmd = 0x19
	LC_RPATH           Cmd = (0x1c | LC_REQ_DYLD)
	LC_CODE_SIGNATURE  Cmd = 0x1d
)
