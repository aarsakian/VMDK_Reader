package main

import (
	"time"

	"flag"
	"fmt"

	"github.com/aarsakian/VMDK_Reader/logger"
)

func main() {
	imagePath := flag.String("image", "", "path to vmdk")
	offset := flag.Int64("offset", -1, "offset to extract data")
	length := flag.Int64("length", -1, "lenght of data")
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
		parentVMDKImage, err := vmdkImage.LocateParent()
		if err != nil {
			logger.VMDKlogger.Error(err)
		} else {
			parentVMDKImage.Process()
			vmdkImage.parentImage = &parentVMDKImage
		}
	}

	if *size {
		totalSize := vmdkImage.GetHDSize()
		fmt.Printf("Virtual disk size is %d GB\n", totalSize/(1024*1000*1000))
	}

	if *length != -1 && *offset != -1 {
		data := vmdkImage.RetrieveData(*offset, *length)
		fmt.Printf("%x\n", data)
	}
}
