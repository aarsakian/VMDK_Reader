package main

import (
	"path/filepath"
	"time"

	"flag"
	"fmt"

	"github.com/aarsakian/VMDK_Reader/extent"
	"github.com/aarsakian/VMDK_Reader/logger"
)

// Descriptor
type Header struct {
	Signature          string
	Version            string
	Encoding           string
	CID                int32
	ParentCID          int32
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
	var extents extent.Extents
	if *imagePath != "" {
		extents = extent.ProcessExtents(*imagePath)
	}

	if *size {
		totalSize := extents.GetHDSize()
		fmt.Printf("Virtual disk size is %d GB\n", totalSize/(1024*1000*1000))
	}

	if *lenght != -1 && *offset != -1 {
		data := extents.RetrieveData(filepath.Dir(*imagePath), *offset, *lenght)
		fmt.Printf("%x\n", data)
	}
}
