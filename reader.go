package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flag"
	"fmt"

	"github.com/aarsakian/VMDK_Reader/extent"
	"github.com/aarsakian/VMDK_Reader/logger"
)

type VMDKImage struct {
	extents extent.Extents
	header  *Header
	path    string
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

func main() {
	imagePath := flag.String("image", "", "path to vmdk")
	offset := flag.Int64("offset", -1, "offset to extract data")
	lenght := flag.Int64("length", -1, "lenght of data")
	size := flag.Bool("size", false, "size of virtual hard disk")
	loggerActive := flag.Bool("log", false, "enable logging")

	now := time.Now()
	logfilename := "logs" + now.Format("2006-01-02T15_04_05") + ".txt"
	logger.InitializeLogger(*loggerActive, logfilename)

	flag.Parse()

	vmdkImage := VMDKImage{path: *imagePath}

	if *imagePath != "" {
		vmdkImage.Process()

	}

	if vmdkImage.HasParent() {
		vmdkImage.LocateParent()
	}

	if *size {
		totalSize := vmdkImage.GetHDSize()
		fmt.Printf("Virtual disk size is %d GB\n", totalSize/(1024*1000*1000))
	}

	if *lenght != -1 && *offset != -1 {
		data := vmdkImage.extents.RetrieveData(filepath.Dir(*imagePath), *offset, *lenght)
		fmt.Printf("%x\n", data)
	}
}

func (vmdkImage *VMDKImage) Process() {
	vmdkImage.AddHeader()
	vmdkImage.extents = extent.LocateExtents(vmdkImage.path)
	vmdkImage.extents.Parse(filepath.Dir(vmdkImage.path))
}

func (vmdkImage *VMDKImage) AddHeader() {
	data, err := os.ReadFile(vmdkImage.path)
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
	vmdkImage.header = header

}

func (vmdkImage VMDKImage) GetHDSize() int64 {
	return vmdkImage.extents.GetHDSize()
}

func (vmdkImage VMDKImage) HasParent() bool {
	return vmdkImage.header.ParentCID != "ffffffff"
}

func (vmdkImage VMDKImage) LocateParent() (os.DirEntry, error) {

	files, err := os.ReadDir(filepath.Dir(vmdkImage.path))
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
		data, err := os.ReadFile(f.Name())
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

			if attr == "CID" && vmdkImage.header.ParentCID == content {
				return f, nil
			}
		}

	}
	return nil, errors.New("parent not found")
}
