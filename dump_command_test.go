package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// sampleTime matches the first record of the committed sample chunk
// 1-1693586596.639629330.flb: seconds 0x64F21B84, nanoseconds 0x3A397767.
var sampleTime = time.Unix(1693588356, 976844647)

// captureOutput redirects *target (os.Stdout or os.Stderr) while fn runs and
// returns everything written to it.
func captureOutput(t *testing.T, target **os.File, fn func()) string {
	t.Helper()
	old := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	*target = w
	defer func() { *target = old }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	w.Close()
	return <-done
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := msgpack.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFlbTimeUnmarshalMsgpack(t *testing.T) {
	var ts flbTime
	payload := []byte{0x64, 0xF2, 0x1B, 0x84, 0x3A, 0x39, 0x77, 0x67}
	if err := ts.UnmarshalMsgpack(payload); err != nil {
		t.Fatal(err)
	}
	if !ts.Time.Equal(sampleTime) {
		t.Errorf("got %v, want %v", ts.Time, sampleTime)
	}

	if err := ts.UnmarshalMsgpack([]byte{0x64, 0xF2}); err == nil {
		t.Error("expected error for payload shorter than 8 bytes")
	}
}

func TestFlbTimeRoundTrip(t *testing.T) {
	in := flbTime{sampleTime}
	payload, err := in.MarshalMsgpack()
	if err != nil {
		t.Fatal(err)
	}
	var out flbTime
	if err := out.UnmarshalMsgpack(payload); err != nil {
		t.Fatal(err)
	}
	if !out.Time.Equal(in.Time) {
		t.Errorf("round trip mismatch: got %v, want %v", out.Time, in.Time)
	}
}

func TestEventTime(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want time.Time
		ok   bool
	}{
		{"ext time", &flbTime{sampleTime}, sampleTime, true},
		{"v2 header array", []interface{}{&flbTime{sampleTime}, map[interface{}]interface{}{}}, sampleTime, true},
		{"int64 seconds", int64(1693588356), time.Unix(1693588356, 0), true},
		{"uint32 seconds", uint32(1693588356), time.Unix(1693588356, 0), true},
		{"uint64 seconds", uint64(1693588356), time.Unix(1693588356, 0), true},
		{"int8 fixnum", int8(5), time.Unix(5, 0), true},
		{"float64 seconds", float64(1693588356.5), time.Unix(1693588356, 500000000), true},
		{"nil", nil, time.Time{}, false},
		{"string", "not a time", time.Time{}, false},
		{"bool", true, time.Time{}, false},
		{"wrong-length array", []interface{}{1, 2, 3}, time.Time{}, false},
		{"array without time", []interface{}{"x", "y"}, time.Time{}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := eventTime(c.in)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if ok && !got.Equal(c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestDecodeClassicEvent(t *testing.T) {
	// Legacy shape: [timestamp, record]. Keys must print sorted.
	payload := mustMarshal(t, []interface{}{
		&flbTime{sampleTime},
		map[string]interface{}{"b": "2", "a": "1"},
	})

	stdout := captureOutput(t, &os.Stdout, func() { decode(payload) })

	want := fmt.Sprintf("[0] [%s, {\"a\": 1, \"b\": 2}]\n", sampleTime.String())
	if stdout != want {
		t.Errorf("got %q, want %q", stdout, want)
	}
}

func TestDecodeModernEvent(t *testing.T) {
	// Fluent Bit >= 2.1 shape: [[timestamp, metadata], record].
	payload := mustMarshal(t, []interface{}{
		[]interface{}{&flbTime{sampleTime}, map[string]interface{}{}},
		map[string]interface{}{"log": "hello"},
	})

	stdout := captureOutput(t, &os.Stdout, func() { decode(payload) })

	want := fmt.Sprintf("[0] [%s, {\"log\": hello}]\n", sampleTime.String())
	if stdout != want {
		t.Errorf("got %q, want %q", stdout, want)
	}
}

func TestDecodeIntegerTimestamp(t *testing.T) {
	payload := mustMarshal(t, []interface{}{
		int64(1693588356),
		map[string]interface{}{"k": "v"},
	})

	stdout := captureOutput(t, &os.Stdout, func() { decode(payload) })

	want := fmt.Sprintf("[0] [%s, {\"k\": v}]\n", time.Unix(1693588356, 0).String())
	if stdout != want {
		t.Errorf("got %q, want %q", stdout, want)
	}
}

func TestDecodeMultipleEvents(t *testing.T) {
	// Chunks hold back-to-back msgpack objects; mix both event shapes.
	payload := append(
		mustMarshal(t, []interface{}{&flbTime{sampleTime}, map[string]interface{}{"n": "0"}}),
		mustMarshal(t, []interface{}{
			[]interface{}{&flbTime{sampleTime.Add(time.Second)}, map[string]interface{}{}},
			map[string]interface{}{"n": "1"},
		})...,
	)

	stdout := captureOutput(t, &os.Stdout, func() { decode(payload) })

	want := fmt.Sprintf("[0] [%s, {\"n\": 0}]\n[1] [%s, {\"n\": 1}]\n",
		sampleTime.String(), sampleTime.Add(time.Second).String())
	if stdout != want {
		t.Errorf("got %q, want %q", stdout, want)
	}
}

func TestDecodeInvalidTimestamp(t *testing.T) {
	payload := mustMarshal(t, []interface{}{true, map[string]interface{}{"k": "v"}})

	stdout := captureOutput(t, &os.Stdout, func() { decode(payload) })

	if !strings.Contains(stdout, "time provided invalid, defaulting to now.") {
		t.Errorf("missing invalid-time warning in %q", stdout)
	}
	if !strings.Contains(stdout, "{\"k\": v}]") {
		t.Errorf("record should still print, got %q", stdout)
	}
}

func TestDecodeNonEventPayload(t *testing.T) {
	// Traces/metrics chunks hold a top-level map, not log events.
	payload := mustMarshal(t, map[string]interface{}{"resourceSpans": []interface{}{}})

	var stdout string
	stderr := captureOutput(t, &os.Stderr, func() {
		stdout = captureOutput(t, &os.Stdout, func() { decode(payload) })
	})

	if stdout != "" {
		t.Errorf("expected no records, got %q", stdout)
	}
	if !strings.Contains(stderr, "unexpected event format") {
		t.Errorf("missing diagnostic in %q", stderr)
	}
}

func TestDecodeInvalidRecord(t *testing.T) {
	payload := mustMarshal(t, []interface{}{&flbTime{sampleTime}, "not a map"})

	var stdout string
	stderr := captureOutput(t, &os.Stderr, func() {
		stdout = captureOutput(t, &os.Stdout, func() { decode(payload) })
	})

	if stdout != "" {
		t.Errorf("expected no records, got %q", stdout)
	}
	if !strings.Contains(stderr, "unexpected record format") {
		t.Errorf("missing diagnostic in %q", stderr)
	}
}

func TestDecodeGarbage(t *testing.T) {
	var stdout string
	stderr := captureOutput(t, &os.Stderr, func() {
		stdout = captureOutput(t, &os.Stdout, func() { decode([]byte{0xC1}) }) // 0xC1 is unused in msgpack
	})

	if stdout != "" {
		t.Errorf("expected no records, got %q", stdout)
	}
	if !strings.Contains(stderr, "stopping at record 0") {
		t.Errorf("missing diagnostic in %q", stderr)
	}
}

func TestDecodeEmpty(t *testing.T) {
	stdout := captureOutput(t, &os.Stdout, func() { decode(nil) })
	if stdout != "" {
		t.Errorf("expected no output, got %q", stdout)
	}
}

func TestDumpSyntheticChunk(t *testing.T) {
	userData := append(
		mustMarshal(t, []interface{}{&flbTime{sampleTime}, map[string]interface{}{"n": "0"}}),
		mustMarshal(t, []interface{}{
			[]interface{}{&flbTime{sampleTime.Add(time.Second)}, map[string]interface{}{}},
			map[string]interface{}{"n": "1"},
		})...,
	)

	// Header + zero CRC + padding + zero metadata length, then user data.
	chunk := make([]byte, FileMetaBytesQuantity-MetadataHeader)
	copy(chunk, ExpectedHeader)
	chunk = append(chunk, userData...)

	dir := t.TempDir()
	chunkPath := filepath.Join(dir, "synthetic.flb")
	if err := os.WriteFile(chunkPath, chunk, 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "dump.out")

	stdout := captureOutput(t, &os.Stdout, func() {
		if err := Dump(DumpOption{FileName: chunkPath, Output: outPath}); err != nil {
			t.Error(err)
		}
	})

	// Check() prints its "Filename ... OK" line before the records.
	want := fmt.Sprintf("[0] [%s, {\"n\": 0}]\n[1] [%s, {\"n\": 1}]\n",
		sampleTime.String(), sampleTime.Add(time.Second).String())
	if !strings.HasSuffix(stdout, want) {
		t.Errorf("got %q, want suffix %q", stdout, want)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, userData) {
		t.Errorf("output file mismatch: got %d bytes, want %d", len(got), len(userData))
	}
}

func TestDumpSampleFile(t *testing.T) {
	const sample = "1-1693586596.639629330.flb"
	raw, err := os.ReadFile(sample)
	if os.IsNotExist(err) {
		t.Skipf("sample chunk %s not present (.flb files are gitignored)", sample)
	}
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "dump.out")
	stdout := captureOutput(t, &os.Stdout, func() {
		if err := Dump(DumpOption{FileName: sample, Output: outPath}); err != nil {
			t.Error(err)
		}
	})

	// This sample has no metadata, so user data starts right after the
	// header, CRC, padding, and metadata length field.
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	want := raw[FileMetaBytesQuantity-MetadataHeader:]
	if !bytes.Equal(got, want) {
		t.Errorf("output file mismatch: got %d bytes, want %d", len(got), len(want))
	}

	wantFirst := fmt.Sprintf("[0] [%s, {", sampleTime.String())
	if !strings.Contains(stdout, wantFirst) {
		t.Errorf("first record should start with %q", wantFirst)
	}
	if strings.Contains(stdout, "time provided invalid") {
		t.Error("all timestamps in the sample should decode")
	}
}
