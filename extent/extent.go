package extent

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/aarsakian/VMDK_Reader/logger"
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
		logger.VMDKlogger.Error(fmt.Sprintf("Error reading file: %e", err))
	}
	lines := bytes.Split(data, []byte("\n"))

	extentDescriptionLocated := false
	diskDatabaseLocated := false
	re := regexp.MustCompile(`([\w+]+)\s([\w+]+)\s([\w+]+).*"([A-Za-z\s\-0-9\.]+)`)
	for idx, line := range lines {
		if idx == 0 && !bytes.Equal(line, []byte("# Disk DescriptorFile")) {
			fmt.Printf("Signature not found %s\n", line)
			logger.VMDKlogger.Warning(fmt.Sprintf("Signature not found %s", line))
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
			matches := re.FindAllSubmatch(line, -1)
			if len(matches) == 0 {
				continue
			}

			cols := matches[0]
			if len(cols) < 5 {
				continue
			}
			nofsectors, e := strconv.Atoi(string(cols[2]))
			if e != nil {
				fmt.Println(e)
				logger.VMDKlogger.Error(fmt.Sprintf("%e", e))
				continue
			}
			extent_ := Extent{AccessMode: string(cols[1]),
				NofSectors: int64(nofsectors),
				ExtentType: string(cols[3]),
				Filename:   string(cols[4])}

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
		logger.VMDKlogger.Error(fmt.Sprintf("Error opening file %s", fullPath))
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
			if startGrainId >= len(extent.GrainOffsets) {
				return remainingDataLen
			}
			grainOffset := int64(extent.GrainOffsets[startGrainId]) * 512
			logger.VMDKlogger.Info(fmt.Sprintf("Reading from %d.",
				grainOffset+startOffsetWithinGrain))

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
		logger.VMDKlogger.Error(fmt.Sprintf("File %s Error reading at %s.",
			extent.Filename, err))
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
	fmt.Printf("Parsing Extents\n")
	for idx := range extents {
		logger.VMDKlogger.Info(fmt.Sprintf("Parsing extent %s.", extents[idx].Filename))
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
	dataBuf := bytes.NewBuffer(make([]byte, 0, length))

	extentEndSector := int64(0)
	offsetBByExtent := offsetB // in extent offset
	//var buf bytes.Buffer
	for _, extent := range extents {
		extentEndSector += extent.NofSectors
		if offsetB > extentEndSector*512 { /// go to next extent
			offsetBByExtent = offsetB - extentEndSector*512
			continue
		}
		logger.VMDKlogger.Info(fmt.Sprintf("Located extent %s", extent.Filename))
		extent.CreateHandle(basepath)
		defer extent.CloseHandler()

		length = extent.LocateData(dataBuf, offsetBByExtent, length)
		// partially filled buffer continue to next extent

		//		offsetB = extentEndSector * 512 ?? need to check
		if length <= 0 {
			break
		}

	}
	return dataBuf.Bytes()
}
