package macho

type Header32 struct {
	Type       uint32
	CpuType    uint32
	CpuSubtype uint32
	Filetype   Filetype
	NCmds      uint32
	SizeOfCmds uint32
	Flags      uint32
}

type Header64 struct {
	Type       uint32
	CpuType    uint32
	CpuSubtype uint32
	Filetype   Filetype
	NCmds      uint32
	SizeOfCmds uint32
	Flags      uint32
	Reserved   uint32
}

type LoadHeader struct {
	Cmd  Cmd
	Size uint32
}

type Load struct {
	LoadHeader
	Off int64
	Raw []byte
}

type Segment struct {
	SegName    [16]byte
	VMAddr     uint32
	VMSize     uint32
	FileOffset uint32
	FileSize   uint32
	MaxProt    int32
	InitProt   int32
	NSect      uint32
	Flags      uint32
}

type Segment64 struct {
	SegName    [16]byte
	VMAddr     uint64
	VMSize     uint64
	FileOffset uint64
	FileSize   uint64
	MaxProt    int32
	InitProt   int32
	NSect      uint32
	Flags      uint32
}

type Symtab struct {
	SymOff  uint32
	NSyms   uint32
	StrOff  uint32
	StrSize uint32
}

type LCCode struct {
	DataOff  uint32
	DataSize uint32
}

type Dylib struct {
	Name                 uint32
	Timestamp            uint32
	CurrentVersion       uint32
	CompatibilityVersion uint32
}

type Rpath struct {
	Name uint32
}
