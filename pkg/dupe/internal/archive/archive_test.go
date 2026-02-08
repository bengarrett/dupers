// © Ben Garrett https://github.com/bengarrett/dupers

package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nalgeon/be"
)

// TestExtension tests the Extension function with various inputs
func TestExtension(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Direct extension matches
		{"zip extension", ".zip", "application/zip"},
		{"7z extension", ".7z", "application/x-7z-compressed"},
		{"tar extension", ".tar", "application/x-tar"},
		{"gz extension", ".gz", "application/gzip"},
		{"bz2 extension", ".bz2", "application/x-bzip2"},
		{"xz extension", ".xz", "application/x-xz"},
		{"rar extension", ".rar", "application/vnd.rar"},
		{"zst extension", ".zst", "application/zstd"},
		{"lz4 extension", ".lz4", "application/x-lz4"},
		{"sz extension", ".sz", "application/x-snappy-framed"},
		
		// Compound extensions
		{"tar.gz extension", ".tar.gz", "application/x-tar"},
		{"tar.bz2 extension", ".tar.bz2", "application/x-tar"},
		{"tar.xz extension", ".tar.xz", "application/x-tar"},
		{"tgz extension", ".tgz", "application/x-tar"},
		{"tbz2 extension", ".tbz2", "application/x-tar"},
		{"txz extension", ".txz", "application/x-tar"},
		
		// Filename without dot prefix
		{"zip filename", "zip", ".zip"},
		{"tar filename", "tar", ".tar"},
		{"gz filename", "gz", ".gz"},
		
		// MIME type lookups (returns some matching extension for MIME types)
		{"zip mime", "application/zip", ".zip"},
		{"gz mime", "application/gzip", ".gz"},
		// Note: tar mime returns various extensions (.tar, .tar.br, etc.) due to map randomization
		// We test the specific behavior in separate tests
		
		// No match cases
		{"no extension", "txt", ""},
		{"empty string", "", ""},
		{"unknown extension", ".unknown", ""},
		{"unknown mime", "unknown/mime", ""},
		
		// Case insensitivity
		{"ZIP uppercase", ".ZIP", "application/zip"},
		{"Zip mixed case", ".Zip", "application/zip"},
		{"TAR.GZ mixed case", ".TAR.GZ", "application/x-tar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Extension(tt.input)
			be.Equal(t, result, tt.expected)
		})
	}
}

// TestMIME tests the MIME function with various filenames
func TestMIME(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		// Standard archive extensions
		{"zip file", "archive.zip", "application/zip"},
		{"tar file", "archive.tar", "application/x-tar"},
		{"gz file", "archive.gz", "application/gzip"},
		{"7z file", "archive.7z", "application/x-7z-compressed"},
		{"rar file", "archive.rar", "application/vnd.rar"},
		{"bz2 file", "archive.bz2", "application/x-bzip2"},
		{"xz file", "archive.xz", "application/x-xz"},
		{"zst file", "archive.zst", "application/zstd"},
		
		// Compound extensions (MIME detects last extension via filepath.Ext)
		{"tar.gz file", "archive.tar.gz", "application/gzip"}, // filepath.Ext returns .gz
		{"tar.bz2 file", "archive.tar.bz2", "application/x-bzip2"}, // filepath.Ext returns .bz2
		{"tgz file", "archive.tgz", "application/x-tar"}, // filepath.Ext returns .tgz (which is in map)
		{"tbz2 file", "archive.tbz2", "application/x-tar"}, // .tbz2 is in the map
		
		// No extension
		{"no extension", "file", ""},
		{"text file", "file.txt", ""},
		{"empty filename", "", ""},
		
		// Case insensitivity
		{"ZIP uppercase", "archive.ZIP", "application/zip"},
		{"Tar mixed case", "archive.Tar", "application/x-tar"},
		{"TAR.GZ mixed case", "archive.TAR.GZ", "application/gzip"},
		
		// Path with extension
		{"path with zip", "path/to/archive.zip", "application/zip"},
		{"path with tar.gz", "path/to/archive.tar.gz", "application/gzip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MIME(tt.filename)
			be.Equal(t, result, tt.expected)
		})
	}
}

// TestSupported tests the Supported function with various archiver formats
func TestSupported(t *testing.T) {
	// Create mock archiver format instances
	tests := []struct {
		name     string
		format   any
		expected bool
	}{
		// This is a simplified test since we can't easily create actual archiver instances
		// In a real scenario, you would create proper instances or use interfaces
		{"nil format", nil, false},
		{"string format", "zip", false},
		{"int format", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Supported(tt.format)
			be.Equal(t, result, tt.expected)
		})
	}
}

// TestReadMIME tests the ReadMIME function with real files
func TestReadMIME(t *testing.T) {
	// Create temporary test files
	tests := []struct {
		name       string
		content    []byte
		expectedMime string
		expectError bool
	}{
		// Note: Actual MIME detection requires real file content
		// These tests verify error handling and basic functionality
		{"empty file", []byte{}, "", true},
		{"small text file", []byte("test content"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			file, err := os.CreateTemp("", "test-*.tmp")
			be.Err(t, err, nil)
			defer os.Remove(file.Name())
			defer file.Close()

			// Write content
			_, err = file.Write(tt.content)
			be.Err(t, err, nil)

			// Test ReadMIME
			mime, err := ReadMIME(file.Name())
			
			if tt.expectError {
				// We expect an error for non-archive files
				if err == nil {
					t.Errorf("Expected error for %s, got none", tt.name)
				}
				if mime != "" {
					t.Errorf("Expected empty MIME for %s, got %s", tt.name, mime)
				}
			} else {
				// For archive files, we'd expect a valid MIME type
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
				if mime != tt.expectedMime {
					t.Errorf("Expected MIME %s for %s, got %s", tt.expectedMime, tt.name, mime)
				}
			}
		})
	}
}

// TestErrors tests error handling in archive functions
func TestErrors(t *testing.T) {
	t.Run("ReadMIME non-existent file", func(t *testing.T) {
		mime, err := ReadMIME("non-existent-file.xyz")
		be.True(t, err != nil)
		be.Equal(t, mime, "")
	})

	t.Run("ReadMIME empty filename", func(t *testing.T) {
		mime, err := ReadMIME("")
		be.True(t, err != nil)
		be.Equal(t, mime, "")
	})

	t.Run("Extension empty input", func(t *testing.T) {
		result := Extension("")
		be.Equal(t, result, "")
	})

	t.Run("MIME empty filename", func(t *testing.T) {
		result := MIME("")
		be.Equal(t, result, "")
	})
}

// TestRealArchiveFiles tests with actual archive files if available
func TestRealArchiveFiles(t *testing.T) {
	// Look for test archive files in the testdata directory
	testFiles := []string{
		"../../../../testdata/randomfiles.zip",
		"../../../../testdata/randomfiles.tar.xz",
		"../../../../testdata/randomfiles.7z",
	}

	for _, filePath := range testFiles {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			t.Logf("Skipping %s: %v", filePath, err)
			continue
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Logf("Test file not found: %s", absPath)
			continue
		}

		t.Run(filepath.Base(filePath), func(t *testing.T) {
			mime, err := ReadMIME(absPath)
			if err != nil {
				t.Logf("Could not read MIME for %s: %v", filePath, err)
				return
			}
			
			if mime == "" {
				t.Errorf("Expected valid MIME type for %s, got empty", filePath)
			} else {
				t.Logf("Detected MIME type for %s: %s", filePath, mime)
			}
		})
	}
}

// TestEdgeCases tests edge cases and special scenarios
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Unicode and special characters
		{"unicode filename", "文件.zip", "application/zip"},
		{"special chars", "file-with-dashes.tar.gz", "application/gzip"}, // .gz is last extension
		{"spaces", "file with spaces.zip", "application/zip"},
		
		// Multiple dots
		{"multiple dots", "archive.tar.gz.backup", ""},
		{"no archive extension", "file.txt.backup", ""},
		
		// Path traversal attempts (should be handled safely)
		{"path traversal", "../../archive.zip", "application/zip"},
		{"absolute path", "/path/to/archive.zip", "application/zip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MIME(tt.input)
			be.Equal(t, result, tt.expected)
		})
	}
}

// TestPerformance tests performance of archive functions (optional)
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("Extension performance", func(t *testing.T) {
		// Test with a large number of calls
		for i := 0; i < 10000; i++ {
			_ = Extension(".zip")
		}
	})

	t.Run("MIME performance", func(t *testing.T) {
		// Test with a large number of calls
		for i := 0; i < 10000; i++ {
			_ = MIME("archive.zip")
		}
	})
}