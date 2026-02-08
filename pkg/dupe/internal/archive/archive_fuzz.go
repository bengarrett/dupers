// © Ben Garrett https://github.com/bengarrett/dupers

//go:build go1.18

package archive

import (
	"testing"
)

// FuzzExtension fuzz tests the file extension detection logic.
func FuzzExtension(f *testing.F) {
	// Add some initial test cases
	testCases := []string{
		"file.txt",
		"archive.zip",
		"document.pdf",
		"image.jpg",
		"noextension",
		".hiddenfile",
		"file.with.multiple.dots.tar.gz",
		"unicode📁file.txt",
		"",
		"a",
		"path/to/file.ext",
		"UPPERCASE.EXT",
		"MixedCase.Ext",
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, filename string) {
		// This should not panic
		ext := Extension(filename)

		// Basic validation - extension should not contain path separators
		if ext != "" {
			// Extension should be a valid file extension format
			if ext[0] != '.' {
				t.Fatalf("Extension should start with '.', got: %q", ext)
			}
		}
	})
}

// FuzzMIME fuzz tests the MIME type detection.
func FuzzMIME(f *testing.F) {
	// Add some initial test cases
	testCases := []string{
		"file.txt",
		"archive.zip",
		"document.pdf",
		"image.jpg",
		"unknown.ext",
		"",
		"path/to/file.txt",
		"file.ZIP", // Test case sensitivity
		"file.TXT",
		"archive.tar.gz",
		"archive.TAR.GZ",
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, filename string) {
		// This should not panic
		mime := MIME(filename)

		// MIME type should be a valid string (could be empty for unknown types)
		// We just ensure it doesn't panic
		_ = mime
	})
}

// FuzzReadMIME fuzz tests the MIME type reading from files.
func FuzzReadMIME(f *testing.F) {
	// Add some initial test cases
	testCases := []string{
		"test.txt",
		"archive.zip",
		"document.pdf",
		"",
		"nonexistent.ext",
		"path/to/file.txt",
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, filename string) {
		// This should not panic
		mime, err := ReadMIME(filename)
		// We expect either a valid MIME type or an error, but not a panic
		if err != nil {
			// Some errors are expected (e.g., file not found)
			return
		}

		// If no error, MIME should not be empty
		if mime == "" {
			t.Fatal("MIME type should not be empty when no error is returned")
		}
	})
}
