package main

import (
	"path/filepath"

	"flag"
	"fmt"

	"github.com/aarsakian/VMDK_Reader/extent"
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
