package extent

type SectorType uint64

// gdOffset points to the level 0 of metadata. It is expressed in sectors.
// overHead is the number of sectors occupied by the metadata.
// numGTEsPerGT is the number of entries in a grain table.
// Sparse Header (512) + Embedded Descriptor (0?) + Redundant Grain dir (0?) + Redundant Grain table #0->#n (2kb for each table 512*32) +
// Grain Dir  + Grain table #0->#n + Padding + Grain entries

//Grain Dir entry points to offset (in sectors) of a grain table
//Grain table entry point to the offset of a grain in sectors
//Grain block of sectors

type SparseHeader struct {
	MagicNumber        string
	Version            uint32
	Flags              uint32
	Capacity           SectorType //extent capacity
	GrainSize          SectorType
	DescriptorOffset   SectorType
	DescriptorSize     SectorType
	NumGTEsPerGT       uint32
	RgdOffset          SectorType // redudant
	GdOffset           SectorType
	OverHead           SectorType
	UncleanShutdown    bool
	SingleEndLineChar  byte
	NonEndLineChar     byte
	DoubleEndLineChar1 byte
	DoubleEndLineChar2 byte
	CompressAlgorithm  uint16
	Pad                [433]byte
}
