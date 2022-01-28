package main

import (
	"encoding/binary"
	"flag"
	"log"
	"os"
)

const HeaderBytesQuantity = 2
const CRCBytesQuantity = 4
const CRCPaddingBytesQuantity = 16
const MetadataLengthBytesQuantity = 2
const FileMetaBytesQuantity = HeaderBytesQuantity + CRCBytesQuantity + CRCPaddingBytesQuantity + MetadataLengthBytesQuantity
const MinRequiredFileLength = FileMetaBytesQuantity + 1

func main() {

	fileName := flag.String("f", "chunk.flb", "File to be processed")
	verbose := flag.Bool("v", false, "Activates verbose mode")
	flag.Parse()

	if *verbose {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(os.Stderr)
	}
	log.Printf("Filename %s\n", *fileName)

	f, err := os.Open(*fileName)

	check(err)
	defer f.Close()

	fileSize := displayFileInfo(f)

	if fileSize < MinRequiredFileLength {
		log.Println("File seems corrupted. Aborting")
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
}

func readUserData(f *os.File, metadataLength uint16, fileSize int64) {
	remainingBytes := fileSize - int64(FileMetaBytesQuantity+metadataLength)
	bytesRead, content := readNBytesFromFile(f, remainingBytes)
	log.Printf("%d bytes from User Content: %s\n", content, string(bytesRead[:content]))
}

func readMetadata(f *os.File, mLength uint16) {
	bytesRead, content := readNBytesFromFile(f, int64(mLength))
	log.Printf("%d bytes from Metadata: %s\n", content, string(bytesRead[:content]))
}

func getMetadataLength(f *os.File) uint16 {
	bytesRead, _ := readNBytesFromFile(f, MetadataLengthBytesQuantity)
	data := binary.BigEndian.Uint16(bytesRead)

	log.Printf("Metadata Length: %d\n", data)
	return data
}

func readPadding(f *os.File) {
	bytesRead, content := readNBytesFromFile(f, CRCPaddingBytesQuantity)
	log.Printf("%d bytes from Padding: %s\n", content, string(bytesRead[:content]))
}

func readCRC(f *os.File) {
	bytesRead, content := readNBytesFromFile(f, CRCBytesQuantity)
	log.Printf("%d bytes from CRC: %s\n", content, string(bytesRead[:content]))
}

func readHeader(f *os.File) ([]byte, int) {
	bytesRead, content := readNBytesFromFile(f, HeaderBytesQuantity)
	log.Printf("%d bytes from header: %s\n", content, string(bytesRead[:content]))
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
	log.Printf("File size: %d bytes\n", fileInformation.Size())
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
