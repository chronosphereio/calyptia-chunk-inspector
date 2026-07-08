package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// flbTime is Fluent Bit's event time: msgpack ext type 0 carrying
// big-endian seconds and nanoseconds, 4 bytes each.
type flbTime struct {
	time.Time
}

func init() {
	msgpack.RegisterExt(0, (*flbTime)(nil))
}

func (t *flbTime) MarshalMsgpack() ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(t.Unix()))
	binary.BigEndian.PutUint32(b[4:], uint32(t.Nanosecond()))
	return b, nil
}

func (t *flbTime) UnmarshalMsgpack(b []byte) error {
	if len(b) != 8 {
		return fmt.Errorf("invalid event time length %d, want 8", len(b))
	}
	sec := binary.BigEndian.Uint32(b)
	nsec := binary.BigEndian.Uint32(b[4:])
	t.Time = time.Unix(int64(sec), int64(nsec))
	return nil
}

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

	decode(userData)

	outputFile, err := os.Create(option.Output)
	check(err)

	_, err = outputFile.Write(userData)
	check(err)
	err = outputFile.Close()
	check(err)

	return nil
}

func decode(userData []byte) {
	dec := msgpack.NewDecoder(bytes.NewReader(userData))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (interface{}, error) {
		return d.DecodeUntypedMap()
	})

	count := 0
	for {
		entry, err := dec.DecodeInterface()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "stopping at record %d: %v\n", count, err)
			break
		}

		event, ok := entry.([]interface{})
		if !ok || len(event) != 2 {
			fmt.Fprintf(os.Stderr, "stopping at record %d: unexpected event format %T\n", count, entry)
			break
		}

		timestamp, ok := eventTime(event[0])
		if !ok {
			fmt.Println("time provided invalid, defaulting to now.")
			timestamp = time.Now()
		}

		record, ok := event[1].(map[interface{}]interface{})
		if !ok {
			fmt.Fprintf(os.Stderr, "stopping at record %d: unexpected record format %T\n", count, event[1])
			break
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
}

// eventTime extracts the event timestamp. Fluent Bit encodes it either as
// ext type 0 (flbTime), a plain integer/float of Unix seconds, or — since
// Fluent Bit 2.1 — wrapped in a two-element header array [timestamp, metadata].
func eventTime(ts interface{}) (time.Time, bool) {
	switch t := ts.(type) {
	case *flbTime:
		return t.Time, true
	case []interface{}:
		if len(t) == 2 {
			return eventTime(t[0])
		}
		return time.Time{}, false
	}
	v := reflect.ValueOf(ts)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return time.Unix(v.Int(), 0), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return time.Unix(int64(v.Uint()), 0), true
	case reflect.Float32, reflect.Float64:
		sec, frac := math.Modf(v.Float())
		return time.Unix(int64(sec), int64(frac*float64(time.Second))), true
	}
	return time.Time{}, false
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
