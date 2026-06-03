package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vmihailenco/msgpack/v5"
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
		fmt.Println("CORRUPTED (file size too small)")
		os.Exit(1)
	}

	if !readHeader(f, verbose) {
		fmt.Println("Corrupted (bad header)")
		os.Exit(1)
	}
	readCRC(f, verbose)
	readPadding(f, verbose)
	metadataLength := getMetadataLength(f, verbose)
	if metadataLength > 0 {
		readChunkType(f, metadataLength, verbose)
	} else {
		if verbose {
			fmt.Println("Chunk Type: logs (assumed, no metadata)")
		}
	}

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

func readChunkType(f *os.File, metadataLength uint16, verbose bool) {
	metadataBytes := readNBytesFromFile(f, int64(metadataLength))

	chunkTypes := map[uint8]string{
		0: "logs",
		1: "metrics",
		2: "traces",
		3: "blobs",
		4: "profiles",
	}

	// First byte of metadata is the event_type
	if len(metadataBytes) > 0 {
		typeByte := metadataBytes[0]
		if chunkType, ok := chunkTypes[typeByte]; ok && verbose {
			fmt.Printf("Chunk Type: %s\n", chunkType)
			return
		}
	}

	if len(metadataBytes) > 0 {
		if metadataBytes[0] == 0x91 && verbose {
			fmt.Println("Chunk Type: logs (legacy, single entry array)")
			return
		}
		if metadataBytes[0] == 0x81 {
			// Possibly a single entry map.
		}
	}

	// Fallback to msgpack unmarshal if the first byte didn't match
	// Some older versions might use a different format
	var metadataMap map[interface{}]interface{}
	err := msgpack.Unmarshal(metadataBytes, &metadataMap)

	if err != nil && verbose {
		fmt.Println("Chunk Type: unknown (could not decode metadata)")
		return
	}

	typeVal, ok := metadataMap["type"]
	if !ok && verbose {
		fmt.Println("Chunk Type: unknown (metadata has no type field)")
		return
	}

	// 1 = logs, 2 = metrics, 3 = traces (old/legacy mapping)
	legacyChunkTypes := map[uint64]string{
		0: "unknown",
		1: "logs",
		2: "metrics",
		3: "traces",
	}

	typeInt, ok := typeVal.(uint64)
	if !ok && verbose {
		fmt.Println("Chunk Type: unknown (type field is not integer)")
		return
	}

	if chunkType, ok := legacyChunkTypes[typeInt]; ok {
		if verbose {
			fmt.Printf("Chunk Type: %s (legacy)\n", chunkType)
		}
	} else {
		if verbose {
			fmt.Printf("Chunk Type: unknown (%d, legacy)\n", typeInt)
		}
	}
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
