package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var verbose *bool

func main() {

	if len(os.Args) < 2 {
		fmt.Println("expected subcommand. Exiting")
		os.Exit(1)
	}
	dumpCmd := flag.NewFlagSet("dump", flag.ExitOnError)
	dumpFlbFile := dumpCmd.String("file", "", "Flb file to be dumped.")
	dumpOutFile := dumpCmd.String("out", "out.json", "Output file. By default out.json")

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	fileName := checkCmd.String("file", "", "File to be processed")
	verbose := checkCmd.Bool("v", false, "Activates verbose mode")
	directory := checkCmd.String("dir", ".", "Directory containing the file(s) to process")

	switch os.Args[1] {
	case "dump":
		err := dumpCmd.Parse(os.Args[2:])
		check(err)
		fmt.Println("subcommand 'dump'")
		fmt.Println("flbFile:", *dumpFlbFile)
		fmt.Println("dumpOutFile:", *dumpOutFile)
		os.Exit(0)
	case "check":
		err := checkCmd.Parse(os.Args[2:])
		check(err)
		options := CheckOption{*fileName, *directory, *verbose}
		err = Check(options)
		check(err)
		os.Exit(0)
	default:
		fmt.Println("Command expected. Quitting")
		os.Exit(1)
	}

	flag.Parse()

	if *verbose {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(os.Stderr)
	}

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
