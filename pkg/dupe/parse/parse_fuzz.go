// © Ben Garrett https://github.com/bengarrett/dupers

//go:build go1.18
// +build go1.18

package parse

import (
	"bytes"
	"os"
	"testing"
)

// createTempFile creates a temporary file with the given data for testing.
func createTempFile(t *testing.T, data []byte) string {
	t.Helper()

	// Create a temporary file
	file, err := os.CreateTemp("", "fuzz-test-*.tmp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()

	// Write the data to the file
	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return file.Name()
}

// FuzzChecksum fuzz tests the checksum calculation to ensure it handles
// various file contents without panicking and produces consistent results.
func FuzzChecksum(f *testing.F) {
	// Add some initial test cases
	testCases := [][]byte{
		[]byte("hello world"),
		[]byte(""),
		[]byte("a"),
		[]byte("\x00\x01\x02\x03\x04\x05"),
		[]byte("The quick brown fox jumps over the lazy dog"),
		bytes.Repeat([]byte("A"), 1024),
		bytes.Repeat([]byte("\x00"), 65536),
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create a temporary file with the fuzz data
		filename := createTempFile(t, data)
		defer os.Remove(filename)

		// This should not panic
		sum, err := Read(filename)
		if err != nil {
			// Some errors are expected (e.g., empty files)
			// but we shouldn't panic
			return
		}

		// Verify the checksum is not zero (unless data is empty)
		if len(data) > 0 && sum == (Checksum{}) {
			t.Fatal("Checksum should not be zero for non-empty data")
		}

		// Verify deterministic output
		sum2, err := Read(filename)
		if err != nil {
			t.Fatalf("Second read failed: %v", err)
		}

		if sum != sum2 {
			t.Fatal("Checksum should be deterministic")
		}
	})
}
