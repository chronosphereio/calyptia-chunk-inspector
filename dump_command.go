package main

import (
	"fmt"
	"os"
)

func Dump(option DumpOption) error {

	if option.FileName == "" {
		fmt.Println("Filename required")
		os.Exit(1)
	}

	err := Check(CheckOption{FileName: option.FileName})
	check(err)

	f, err := os.Open(option.FileName)
	check(err)
	mLength := getMetadataLength(f, false)

	if mLength > 0 {
		readMetadata(f, mLength, false)
	}

	fileSize := fileInfo(f, false)

	output := readUserData(f, mLength, fileSize, false)

	outputFile, err := os.Create(option.Output)
	check(err)

	_, err = outputFile.Write(output)
	check(err)
	err = outputFile.Close()
	check(err)

	return nil
}

func readUserData(f *os.File, metadataLength uint16, fileSize int64, verbose bool) []byte {
	remainingBytes := fileSize - int64(FileMetaBytesQuantity+metadataLength)
	bytesRead, content := readNBytesFromFile(f, remainingBytes)
	output := bytesRead[:content]
	if verbose {
		fmt.Printf("%d bytes read from User Content: %s\n", content, string(output))
	}
	return output
}

func readMetadata(f *os.File, mLength uint16, verbose bool) {
	bytesRead, content := readNBytesFromFile(f, int64(mLength))
	if verbose {
		fmt.Printf("%d bytes read from Metadata: %s\n", content, string(bytesRead[:content]))
	}
}
