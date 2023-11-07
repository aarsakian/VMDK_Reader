package extent

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aarsakian/VMDK_Reader/utils"
)

type Extents []Extent

type Extent struct {
	AccessMode       string
	NofSectors       int64
	ExtentType       string
	Filename         string
	StartSector      int32 //only for flat extents
	PartitionUUID    int64
	DeviceIdentifier int
	Fhandle          *os.File
	SparseHeader     *SparseHeader
	GrainOffsets     GrainOffsets
}

func ProcessExtents(imagePath string) Extents {
	var extents Extents

	data, err := os.ReadFile(imagePath)

	if err != nil {
		fmt.Println("Error reading file:", err)
	}
	lines := bytes.Split(data, []byte("\n"))

	extentDescriptionLocated := false
	diskDatabaseLocated := false

	for idx, line := range lines {
		if idx == 0 && !bytes.Equal(line, []byte("# Disk DescriptorFile")) {
			fmt.Printf("Signature not found %s\n", line)
			os.Exit(0)

		}
		if bytes.Equal(line, []byte("# Extent description")) {
			extentDescriptionLocated = true
			continue
		} else if bytes.Equal(line, []byte("# The Disk Data Base ")) {
			extentDescriptionLocated = false
			diskDatabaseLocated = true
			continue
		}

		if extentDescriptionLocated {
			cols := bytes.Split(line, []byte(" "))
			if len(cols) < 4 {

				continue
			}
			extent_ := Extent{AccessMode: string(cols[0]),
				NofSectors: utils.ReadEndianInt(cols[1]),
				ExtentType: string(cols[2]),
				Filename:   strings.ReplaceAll(string(cols[3]), "\"", "")}

			extents = append(extents, extent_)

		}

		if diskDatabaseLocated {

		}

	}

	extents.Parse(filepath.Dir(imagePath))
	return extents
}

func (extent *Extent) CreateHandle(basepath string) {
	fullPath := path.Join(basepath, extent.Filename)
	file, err := os.Open(fullPath)
	if err != nil {
		fmt.Printf("Error opening file %s\n", fullPath)
	}
	extent.Fhandle = file
}

func (extent Extent) CloseHandler() {
	extent.Fhandle.Close()
}

func (extent Extent) LocateData(data *bytes.Buffer, offsetB int64, length int64) int64 {
	var remainingDataLen int64
	var buf []byte
	switch extent.ExtentType {
	case "SPARSE":
		grainSizeB := int64(extent.SparseHeader.GrainSize) * 512
		startGrainId := int(offsetB / (int64(extent.SparseHeader.GrainSize) * 512))

		startOffsetWithinGrain := offsetB % (int64(extent.SparseHeader.GrainSize) * 512)
		remainingDataLen = length
		for remainingDataLen > 0 {
			grainOffset := int64(extent.GrainOffsets[startGrainId]) * 512
			fmt.Printf("Reading from %d\n", grainOffset+startOffsetWithinGrain)

			if remainingDataLen < grainSizeB {
				buf = make([]byte, remainingDataLen)

			} else {
				buf = make([]byte, grainSizeB-startOffsetWithinGrain)

			}

			//otherwise is zero
			if grainOffset != 0 {

				extent.ReadAt(buf, grainOffset+startOffsetWithinGrain)
			}

			data.Write(buf)
			remainingDataLen -= int64(len(buf))

			startGrainId += 1
			startOffsetWithinGrain = 0 // next grain is always from beginning
		}

	}
	return remainingDataLen

}

func (extent Extent) ReadAt(buf []byte, offset int64) {
	_, err := extent.Fhandle.ReadAt(buf, offset)
	if err != nil {
		fmt.Printf("File %s Error reading at %s\n", extent.Filename, err)
	}
}

func (extents Extents) GetHDSize() int64 {
	totalSize := int64(0)
	for _, extent := range extents {
		totalSize += int64(extent.NofSectors)
	}
	return int64(totalSize) * 512
}

func (extents Extents) Parse(basepath string) {
	for idx := range extents {
		fmt.Printf("Parsing extent %s\n", extents[idx].Filename)
		extents[idx].CreateHandle(basepath)
		defer extents[idx].CloseHandler()
		switch extents[idx].ExtentType {
		case "SPARSE":
			var sparseH *SparseHeader = new(SparseHeader)
			buf := make([]byte, 512)
			extents[idx].ReadAt(buf, 0)
			utils.Unmarshal(buf, sparseH)
			extents[idx].SparseHeader = sparseH
			buf = make([]byte, sparseH.OverHead*512)
			extents[idx].ReadAt(buf, int64(sparseH.GdOffset)*512)

			extents[idx].GrainOffsets = sparseH.PopulateGrainOffsets(buf)
		}
	}

}

func (extents Extents) RetrieveData(basepath string, offsetB int64, length int64) []byte {
	dataBuf := bytes.Buffer{}
	dataBuf.Grow(int(length))

	extentEndSector := int64(0)
	//var buf bytes.Buffer
	for _, extent := range extents {
		extentEndSector += extent.NofSectors
		if offsetB > extentEndSector { /// go to next extent
			continue
		}
		fmt.Printf("Located extent %s\n", extent.Filename)
		extent.CreateHandle(basepath)
		defer extent.CloseHandler()

		length = extent.LocateData(&dataBuf, offsetB, length)
		// partially filled buffer continue to next extent

		offsetB = extentEndSector * 512
		if length <= 0 {
			break
		}

	}
	return dataBuf.Bytes()
}