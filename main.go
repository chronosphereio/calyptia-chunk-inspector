package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("expected subcommand. Exiting")
		os.Exit(1)
	}
	dumpCmd := flag.NewFlagSet("dump", flag.ExitOnError)
	dumpFlbFile := dumpCmd.String("file", "", "Flb file to be dumped.")
	dumpOutFile := dumpCmd.String("out", "dump.out", "Output file.")
	dumpVerbose := dumpCmd.Bool("v", false, "Activates verbose mode.")

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	fileName := checkCmd.String("file", "", "File to be processed. Mutually exclusive with -dir.")
	verbose := checkCmd.Bool("v", false, "Activates verbose mode.")
	directory := checkCmd.String("dir", ".", "Directory containing the file(s) to process. Mutually exclusive with -file.")

	switch os.Args[1] {
	case "dump":
		err := dumpCmd.Parse(os.Args[2:])
		check(err)
		options := DumpOption{*dumpFlbFile, *dumpOutFile, *dumpVerbose}
		err = Dump(options)
		check(err)
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
