package extent

import (
	"github.com/aarsakian/VMDK_Reader/utils"
)

type GrainOffsets []int

//grain size 2^G sectors  a block of sectors containing data for the virtual di
//gtCoverage by a grain table nof GTE x grainSize
//he size of a sparse extent should be a multiple of grainSize
//each gt is 2kb 512 entries of 32 bit
//GDE offset in sectors of a grain table in a sparse extent
//GTE offset of a grain in the sparse extent

func (sparseH SparseHeader) PopulateGrainOffsets(data []byte) GrainOffsets {
	var gteOffsets GrainOffsets
	var gtOffset int
	var gteOffset int
	var startOffset int
	gtCoverage := sparseH.NumGTEsPerGT * uint32(sparseH.GrainSize) //covered sectors by a single grain table
	nofGDEntries := int(sparseH.Capacity) / int(gtCoverage)        //required GD entries

	for gdEntry := 0; gdEntry < nofGDEntries; gdEntry++ {
		gtOffset = int(utils.ReadEndianInt(data[gdEntry*4 : gdEntry*4+4])) //grain table offset

		for gtEntry := 0; gtEntry < int(sparseH.NumGTEsPerGT); gtEntry++ {
			startOffset = gtOffset*512 + gtEntry*4 - int(sparseH.GdOffset)*512 //offset to buffer
			gteOffset = int(utils.ReadEndianInt(data[startOffset : startOffset+4]))
			//fmt.Printf("%d %d | %d\t", counter, gtOffset*512+gtEntry*4, gteOffset)
			gteOffsets = append(gteOffsets, gteOffset)

		}
	}
	return gteOffsets
}

func (g GrainOffsets) Less(i, j int) bool {
	return g[i] < g[j]
}

func (g GrainOffsets) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func (g GrainOffsets) Len() int {
	return len(g)
}
