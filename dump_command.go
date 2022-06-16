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
	var ts interface{}
	var record map[interface{}]interface{}

	decoder := output.NewDecoder(data, length)
	if decoder == nil {
		fmt.Errorf("dec is nil")
	}

	ret, ts, record := output.GetRecord(decoder)
	if ret != 0 {
		print("No more records")
	}

	//recordu := msgpack.Unmarshal(userData, ts)
	//fmt.Printf("Unmarshaled :%s\n", ts)
	//fmt.Printf("not :%s\n", recordu)

	//var dummyRecord = [29]byte{0x92, /* fix array 2 */
	//	0xd7, 0x00, 0x5e, 0xa9, 0x17, 0xe0, 0x00, 0x00, 0x00, 0x00, /* 2020/04/29 06:00:00*/
	//	0x82,                                           /* fix map 2*/
	//	0xa7, 0x63, 0x6f, 0x6e, 0x70, 0x61, 0x63, 0x74, /* fix str 7 "compact" */
	//	0xc3,                                     /* true */
	//	0xa6, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, /* fix str 6 "schema" */
	//	0x01, /* fix int 1 */
	//}

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
		fmt.Printf("\"%s\": %v, ", k, v)
	}
	fmt.Printf("}\n")

	return 0

}

func readUserData(f *os.File, metadataLength uint16, fileSize int64, verbose bool) []byte {
	userDataStart := int64(FileMetaBytesQuantity - MetadataHeader + metadataLength)
	remainingBytes := fileSize - userDataStart
	f.Seek(userDataStart, 0)
	fmt.Printf("User data starts in byte: %d\nRemaining bytes: %d\n", userDataStart, remainingBytes)
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
