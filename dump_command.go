package main

import "C"
import (
	"fmt"
	"github.com/fluent/fluent-bit-go/output"
	"os"
	"time"
	"unsafe"
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
	mLength := getMetadataLength(f, option.Verbose)

	if mLength > 0 {
		readMetadata(f, mLength, option.Verbose)
	}

	fileSize := fileInfo(f, option.Verbose)

	userData := readUserData(f, mLength, fileSize, option.Verbose)

	thePointer := unsafe.Pointer(&userData)
	decode(thePointer, len(userData))

	outputFile, err := os.Create(option.Output)
	check(err)

	_, err = outputFile.Write(userData)
	check(err)
	err = outputFile.Close()
	check(err)

	return nil
}

func decode(data unsafe.Pointer, length int) int {
	decoder := output.NewDecoder(data, length)
	if decoder == nil {
		fmt.Errorf("dec is nil")
	}

	for {
		var ts interface{}
		var record map[interface{}]interface{}

		ret, ts, record := output.GetRecord(decoder)
		if ret != 0 { // No more records
			break
		}

		var timestamp time.Time

		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			fmt.Println("time provided invalid, defaulting to now.")
			timestamp = time.Now()
		}

		// Print record keys and values
		fmt.Printf("[%s, {", timestamp.String())
		for k, v := range record {
			fmt.Printf("\"%s\": %s, ", k, v)
		}
		fmt.Printf("}\n")

	}
	return 0

}

func readUserData(f *os.File, metadataLength uint16, fileSize int64, verbose bool) []byte {
	userDataStart := int64(FileMetaBytesQuantity - MetadataHeader + metadataLength)
	remainingBytes := fileSize - userDataStart
	f.Seek(userDataStart, 0)
	bytesRead, content := readNBytesFromFile(f, remainingBytes)
	userData := bytesRead[:content]
	if verbose {
		fmt.Printf("%d bytes read from User Content: [%s]\n", content, string(userData))
	}
	return bytesRead
}

func readMetadata(f *os.File, mLength uint16, verbose bool) {
	f.Seek(4, 1)                                               //metadata headers
	bytesRead, size := readNBytesFromFile(f, int64(mLength-4)) //metadata headers bytes are part of the metadata declared size
	if verbose {
		fmt.Printf("%d bytes read from Metadata: [%s]\n", size, string(bytesRead[:size]))
	}
}
