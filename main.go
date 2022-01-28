package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	HeaderBytesQuantity         = 2
	CRCBytesQuantity            = 4
	CRCPaddingBytesQuantity     = 16
	MetadataLengthBytesQuantity = 2
	FileMetaBytesQuantity       = HeaderBytesQuantity + CRCBytesQuantity + CRCPaddingBytesQuantity + MetadataLengthBytesQuantity
	MinRequiredFileLength       = FileMetaBytesQuantity + 1
)

var verbose *bool

func main() {

	fileName := flag.String("file", "", "File to be processed")
	verbose = flag.Bool("v", false, "Activates verbose mode")
	directory := flag.String("dir", ".", "Directory containing the file(s) to process")
	flag.Parse()

	if *verbose {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(os.Stderr)
	}

	if *fileName != "" {
		checkFile(*fileName)
		os.Exit(0)
	}

	if *directory != "" {
		_, err := ioutil.ReadDir(*directory)
		check(err)
		filepath.Walk(*directory,
			func(path string, info fs.FileInfo, err error) error {
				if strings.HasSuffix(path, ".flb") {
					checkFile(path)
				}
				return nil
			})
	}
}

func checkFile(fileName string) {
	fmt.Printf("Filename %s\n", fileName)

	f, err := os.Open(fileName)

	check(err)
	defer f.Close()

	fileSize := displayFileInfo(f)

	if fileSize < MinRequiredFileLength {
		fmt.Println("File seems corrupted. Aborting")
		os.Exit(1)
	}

	readHeader(f)

	readCRC(f)

	readPadding(f)

	mLength := getMetadataLength(f)

	if mLength > 0 {
		readMetadata(f, mLength)
	}

	readUserData(f, mLength, fileSize)

	err = f.Close()
	check(err)

	fmt.Println("OK")
}

func readUserData(f *os.File, metadataLength uint16, fileSize int64) {
	remainingBytes := fileSize - int64(FileMetaBytesQuantity+metadataLength)
	bytesRead, content := readNBytesFromFile(f, remainingBytes)
	if *verbose {
		fmt.Printf("%d bytes read from User Content: %s\n", content, string(bytesRead[:content]))
	}
}

func readMetadata(f *os.File, mLength uint16) {
	bytesRead, content := readNBytesFromFile(f, int64(mLength))
	if *verbose {
		fmt.Printf("%d bytes read from Metadata: %s\n", content, string(bytesRead[:content]))
	}
}

func getMetadataLength(f *os.File) uint16 {
	bytesRead, _ := readNBytesFromFile(f, MetadataLengthBytesQuantity)
	data := binary.BigEndian.Uint16(bytesRead)
	if *verbose {
		fmt.Printf("Metadata Length: %d\n", data)
	}
	return data
}

func readPadding(f *os.File) {
	bytesRead, content := readNBytesFromFile(f, CRCPaddingBytesQuantity)
	if *verbose {
		fmt.Printf("%d bytes read from Padding: %s\n", content, string(bytesRead[:content]))
	}
}

func readCRC(f *os.File) {
	bytesRead, content := readNBytesFromFile(f, CRCBytesQuantity)
	if *verbose {
		fmt.Printf("%d bytes from CRC: %s\n", content, string(bytesRead[:content]))
	}
}

func readHeader(f *os.File) ([]byte, int) {
	bytesRead, content := readNBytesFromFile(f, HeaderBytesQuantity)
	if *verbose {
		fmt.Printf("%d bytes from header: %s\n", content, string(bytesRead[:content]))
	}
	return bytesRead, content
}

func readNBytesFromFile(file *os.File, bytesToRead int64) ([]byte, int) {
	b1 := make([]byte, bytesToRead)
	n1, err := file.Read(b1)
	check(err)
	return b1, n1
}

func displayFileInfo(f *os.File) int64 {
	fileInformation, err := f.Stat()
	check(err)
	if *verbose {
		fmt.Printf("File size: %d bytes\n", fileInformation.Size())
	}
	return fileInformation.Size()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

/**
+--------------+----------------+
|     0xC1     |     0x00       +--> Header 2 bytes
+--------------+----------------+
|    4 BYTES CRC32 + 16 BYTES   +--> CRC32(Content) + Padding
+-------------------------------+
|            Content            |
|  +-------------------------+  |
|  |         2 BYTES         +-----> Metadata Length
|  +-------------------------+  |
|  +-------------------------+  |
|  |                         |  |
|  |        Metadata         +-----> Optional Metadata (up to 65535 bytes)
|  |                         |  |
|  +-------------------------+  |
|  +-------------------------+  |
|  |                         |  |
|  |       Content Data      +-----> User Data
|  |                         |  |
|  +-------------------------+  |
+-------------------------------+

*/
