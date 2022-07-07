# Chunk Inspector

This is a basic tool to handle Fluent Bit's flb (chunk) files.


## Usage

These are the subcommands currently supported:

* [Check](#check)
* [Dump](#dump)

### check

It will check for corrupted/invalid flb files.

```shell
Usage of check:
  -dir string
        Directory containing the file(s) to process (default ".")
  -file string
        File to be processed
  -v    Activates verbose mode
```

```shell
$ chunk-inspector check
Filename 1-1642796665.47813680.flb OK
Filename 1-1642796694.873134717.flb Corrupted
```
```shell
$ chunk-inspector check -v    
Filename 1-1642796665.47813680.flb 
File size: 660 bytes
2 bytes from header: �
4 bytes from CRC: 
16 bytes read from Padding: 
Metadata Length: 122
OK
Filename 1-1642796694.873134717.flb 
File size: 24 bytes
Corrupted
```

### dump

It will dump the user content of the flb file to the output file.

```shell
Usage of dump:
  -file string
        Flb file to be dumped.
  -out string
        Output file. (default "dump.out")

```