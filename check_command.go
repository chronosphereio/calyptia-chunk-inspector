package main

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func Check(option CheckOption) error {
	if option.FileName != "" {
		err := checkFile(option.FileName, option.Verbose)
		check(err)
	} else {
		if option.Directory != "" {
			_, err := ioutil.ReadDir(option.Directory)
			check(err)
			err = filepath.Walk(option.Directory,
				func(path string, info fs.FileInfo, err error) error {
					if strings.HasSuffix(path, ".flb") {
						err = checkFile(path, option.Verbose)
						check(err)
					}
					return nil
				})
			check(err)
		}
	}

	return nil
}

func checkFile(fileName string, verbose bool) error {
	fmt.Printf("Filename %s ", fileName)
	f, err := os.Open(fileName)
	check(err)
	defer f.Close()

	fileSize := fileInfo(f, verbose)

	if fileSize < MinRequiredFileLength {
		fmt.Println("Corrupted")
		os.Exit(1)
	}

	readHeader(f, verbose)
	readCRC(f, verbose)
	readPadding(f, verbose)
	getMetadataLength(f, verbose)

	err = f.Close()
	check(err)

	fmt.Println("OK")
	return nil
}

func fileInfo(f *os.File, verbose bool) int64 {
	fileInformation, err := f.Stat()
	check(err)
	if verbose {
		fmt.Printf("\nFile size: %d bytes\n", fileInformation.Size())
	}
	return fileInformation.Size()
}

func readHeader(f *os.File, verbose bool) ([]byte, int) {
	bytesRead, content := readNBytesFromFile(f, HeaderBytesQuantity)
	if verbose {
		fmt.Printf("%d bytes from header: %s\n", content, string(bytesRead[:content]))
	}
	return bytesRead, content
}

func readCRC(f *os.File, verbose bool) {
	bytesRead, content := readNBytesFromFile(f, CRCBytesQuantity)
	if verbose {
		fmt.Printf("%d bytes from CRC: %s\n", content, string(bytesRead[:content]))
	}
}

func readPadding(f *os.File, verbose bool) {
	bytesRead, content := readNBytesFromFile(f, CRCPaddingBytesQuantity)
	if verbose {
		fmt.Printf("%d bytes read from Padding: %s\n", content, string(bytesRead[:content]))
	}
}

func getMetadataLength(f *os.File, verbose bool) uint16 {
	_, err := f.Seek(MetadataStart, 0)
	check(err)

	bytesRead, _ := readNBytesFromFile(f, MetadataLengthBytesQuantity)
	data := binary.BigEndian.Uint16(bytesRead)
	if verbose {
		fmt.Printf("Metadata Length: %d\n", data)
	}
	return data
}

func readNBytesFromFile(file *os.File, bytesToRead int64) ([]byte, int) {
	b1 := make([]byte, bytesToRead)
	n1, err := file.Read(b1)
	check(err)
	return b1, n1
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
