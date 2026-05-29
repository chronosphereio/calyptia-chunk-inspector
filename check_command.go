package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Check(option CheckOption) error {
	if option.FileName != "" {
		err := checkFile(option.FileName, option.Verbose)
		check(err)
		return nil
	}

	if option.Directory == "" {
		return nil
	}

	err := filepath.Walk(option.Directory,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".flb" {
				err = checkFile(path, option.Verbose)
				check(err)
			}
			return nil
		})
	check(err)
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

	if !readHeader(f, verbose) {
		fmt.Println("Corrupted (bad header)")
		os.Exit(1)
	}
	readCRC(f, verbose)
	readPadding(f, verbose)
	getMetadataLength(f, verbose)

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

func readHeader(f *os.File, verbose bool) bool {
	bytesRead := readNBytesFromFile(f, HeaderBytesQuantity)
	if verbose {
		fmt.Printf("%d bytes from header: % X\n", len(bytesRead), bytesRead)
	}
	return bytes.Equal(bytesRead, ExpectedHeader)
}

func readCRC(f *os.File, verbose bool) {
	bytesRead := readNBytesFromFile(f, CRCBytesQuantity)
	if verbose {
		fmt.Printf("%d bytes from CRC: % X\n", len(bytesRead), bytesRead)
	}
}

func readPadding(f *os.File, verbose bool) {
	bytesRead := readNBytesFromFile(f, CRCPaddingBytesQuantity)
	if verbose {
		fmt.Printf("%d bytes read from Padding: % X\n", len(bytesRead), bytesRead)
	}
}

func getMetadataLength(f *os.File, verbose bool) uint16 {
	_, err := f.Seek(MetadataStart, 0)
	check(err)

	bytesRead := readNBytesFromFile(f, MetadataLengthBytesQuantity)
	data := binary.BigEndian.Uint16(bytesRead)
	if verbose {
		fmt.Printf("Metadata Length: %d\n", data)
	}
	return data
}

func readNBytesFromFile(file *os.File, bytesToRead int64) []byte {
	buf := make([]byte, bytesToRead)
	_, err := io.ReadFull(file, buf)
	check(err)
	return buf
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
