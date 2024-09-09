package vmdk

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aarsakian/VMDK_Reader/extent"
	"github.com/aarsakian/VMDK_Reader/logger"
)

type VMDKImage struct {
	Extents     extent.Extents
	Header      *Header
	Path        string
	ParentImage *VMDKImage
}

// Descriptor
type Header struct {
	Version            string
	Encoding           string
	CID                string
	ParentCID          string
	IsNativeSnaphost   string
	CreateType         string
	ParentFileNameHint string
}

func (vmdkImage *VMDKImage) Process() {
	vmdkImage.AddHeader()
	vmdkImage.Extents = extent.LocateExtents(vmdkImage.Path)
	vmdkImage.Extents.Parse()
}

func (vmdkImage *VMDKImage) AddHeader() {
	data, err := os.ReadFile(vmdkImage.Path)
	header := new(Header)
	if err != nil {
		fmt.Println("Error reading file:", err)
		logger.VMDKlogger.Error(fmt.Sprintf("Error reading file: %e", err))
	}
	lines := bytes.Split(data, []byte("\n"))
	var attr, content string
	for idx, line := range lines {
		if bytes.Contains(line, []byte("=")) {
			attr = strings.Split(string(line), "=")[0]
			content = strings.Split(string(line), "=")[1]
		}

		if idx == 0 && !bytes.Equal(line, []byte("# Disk DescriptorFile")) {
			fmt.Printf("Signature not found %s\n", line)
			logger.VMDKlogger.Warning(fmt.Sprintf("Signature not found %s", line))
			os.Exit(0)

		} else if attr == "version" {
			header.Version = content
		} else if attr == "encoding" {
			header.Encoding = content
		} else if attr == "CID" {
			header.CID = content
		} else if attr == "parentCID" {
			header.ParentCID = content
		} else if attr == "createType" {
			header.CreateType = content
		} else if attr == "parentFileNameHint" {
			header.ParentFileNameHint = content
		} else if bytes.Equal(line, []byte("# Extent description")) {
			break
		}
	}
	vmdkImage.Header = header

}

func (vmdkImage VMDKImage) GetHDSize() int64 {
	return vmdkImage.Extents.GetHDSize()
}

func (vmdkImage VMDKImage) HasParent() bool {
	return vmdkImage.Header.ParentCID != "ffffffff"
}

func (vmdkImage VMDKImage) RetrieveData(offset int64, length int64) []byte {
	var dataBuf, dataParentBuf bytes.Buffer
	dataBuf.Grow(int(length))
	grainOffsets := vmdkImage.Extents.RetrieveData(&dataBuf, offset, length) //needed to check for zeros

	dataParentBuf.Grow(int(length))
	grainSize := vmdkImage.GetGrainSizeB()

	if vmdkImage.HasParent() {
		offsetGrain := grainSize
		for idx, grainOffset := range grainOffsets {
			childBytes := dataBuf.Next(int(grainSize))
			if grainOffset != 0 {
				dataParentBuf.Write(childBytes)
				continue
			}

			offsetGrain *= int64(idx)
			vmdkImage.ParentImage.Extents.RetrieveData(&dataParentBuf, offset-offsetGrain, length-offsetGrain) //retrieve only data for zeroed grain offsetss

		}
		return dataParentBuf.Bytes()
	} else {
		return dataBuf.Bytes()
	}

}

func (vmdkImage VMDKImage) GetGrainSizeB() int64 {
	return vmdkImage.Extents[0].GetGrainSizeB()
}

func (vmdkImage VMDKImage) LocateParent() (VMDKImage, error) {

	files, err := os.ReadDir(filepath.Dir(vmdkImage.Path))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		finfo, err := f.Info()
		if err != nil {
			logger.VMDKlogger.Error(err)
			continue
		}
		if finfo.Size() > 2*1024 || filepath.Ext(f.Name()) != ".vmdk" {
			continue
		}
		parentPathfile := filepath.Join(filepath.Dir(vmdkImage.Path), f.Name())
		data, err := os.ReadFile(parentPathfile)
		if err != nil {
			logger.VMDKlogger.Error(err)
			continue
		}
		lines := bytes.Split(data, []byte("\n"))
		var attr, content string
		for _, line := range lines {
			if bytes.Contains(line, []byte("=")) {
				attr = strings.Split(string(line), "=")[0]
				content = strings.Split(string(line), "=")[1]
			}

			if attr == "CID" && vmdkImage.Header.ParentCID == content {
				return VMDKImage{Path: parentPathfile}, nil
			}
		}

	}
	return VMDKImage{}, errors.New("parent not found")
}
