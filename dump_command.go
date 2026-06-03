package main

import "C"
import (
	"fmt"
	"os"
	"sort"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
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
	defer f.Close()

	mLength := getMetadataLength(f, option.Verbose)

	if mLength > 0 {
		readMetadata(f, mLength, option.Verbose)
	}

	fileSize := fileInfo(f, option.Verbose)

	userData := readUserData(f, mLength, fileSize, option.Verbose)

	thePointer := unsafe.Pointer(C.CBytes(userData))
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
		fmt.Fprintln(os.Stderr, "decoder is nil")
		os.Exit(1)
	}

	count := 0
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

		type kv struct {
			key string
			val interface{}
		}
		entries := make([]kv, 0, len(record))
		for k, v := range record {
			entries = append(entries, kv{fmt.Sprintf("%s", k), v})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })

		fmt.Printf("[%d] [%s, {", count, timestamp.String())
		for i, e := range entries {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("\"%s\": %s", e.key, e.val)
		}
		fmt.Printf("}]\n")
		count++
	}
	return 0

}

func readUserData(f *os.File, metadataLength uint16, fileSize int64, verbose bool) []byte {
	userDataStart := int64(FileMetaBytesQuantity-MetadataHeader) + int64(metadataLength)
	remainingBytes := fileSize - userDataStart
	_, err := f.Seek(userDataStart, 0)
	check(err)
	userData := readNBytesFromFile(f, remainingBytes)
	if verbose {
		fmt.Printf("%d bytes read from User Content: [%s]\n", len(userData), string(userData))
	}
	return userData
}

func readMetadata(f *os.File, mLength uint16, verbose bool) {
	_, err := f.Seek(MetadataHeader, 1) //metadata headers
	check(err)
	bytesRead := readNBytesFromFile(f, int64(mLength-MetadataHeader)) //metadata headers bytes are part of the metadata declared size
	if verbose {
		fmt.Printf("%d bytes read from Metadata: [%s]\n", len(bytesRead), string(bytesRead))
	}
}
