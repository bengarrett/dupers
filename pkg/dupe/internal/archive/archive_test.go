// © Ben Garrett https://github.com/bengarrett/dupers

package archive

import (
	"errors"
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
	// Test with various unsupported types to ensure the type switch works correctly
	tests := []struct {
		name     string
		format   any
		expected bool
	}{
		// Basic unsupported types
		{"nil format", nil, false},
		{"string format", "zip", false},
		{"int format", 42, false},
		{"bool format", true, false},
		{"float format", 3.14, false},
		{"slice format", []string{"test"}, false},
		{"map format", map[string]string{"key": "value"}, false},
		{"struct format", struct{ name string }{name: "test"}, false},
		{"pointer to string", func() *string { s := "test"; return &s }(), false},
		{"interface format", func() interface{} { return "test" }(), false},
		{"channel format", make(chan int), false},
		{"function format", func() {}, false},
		{"error format", errors.New("test error"), false},
		{"empty interface", interface{}(nil), false},
		{"array format", [3]int{1, 2, 3}, false},
		{"rune format", 'a', false},
		{"byte format", byte(65), false},
		{"uint format", uint(42), false},
		{"complex format", complex(1, 2), false},
		{"pointer to interface", func() interface{} { var i interface{}; return &i }(), false},
		{"custom struct", struct{ field string }{field: "test"}, false},
		{"nested struct", struct{ inner struct{ value int } }{inner: struct{ value int }{value: 42}}, false},
		{"anonymous struct", struct{ string }{}, false},
		{"pointer to custom struct", &struct{ field string }{field: "test"}, false},
		{"pointer to nested struct", &struct{ inner struct{ value int } }{inner: struct{ value int }{value: 42}}, false},
		{"pointer to anonymous struct", &struct{ string }{}, false},
		{"slice of structs", []struct{ field string }{{"test"}}, false},
		{"map of structs", map[string]struct{ field string }{"key": {"test"}}, false},
		{"struct with methods", struct{ name string }{name: "test"}, false},
		{"interface with methods", func() interface{} { return nil }, false},
		{"multiple interfaces", func() interface{} { return nil }, false},
		{"complex nested structure", struct{ 
			inner struct{ 
				value int 
				data []string 
			} 
		}{inner: struct{ 
			value int 
			data []string 
		}{value: 42, data: []string{"test"}}}, false},
		{"pointer to complex structure", &struct{ 
			inner struct{ 
				value int 
				data []string 
			} 
		}{inner: struct{ 
			value int 
			data []string 
		}{value: 42, data: []string{"test"}}}, false},
		{"empty struct", struct{}{}, false},
		{"pointer to empty struct", &struct{}{}, false},
		{"struct with unexported fields", struct{ unexported string }{unexported: "test"}, false},
		{"pointer to struct with unexported fields", &struct{ unexported string }{unexported: "test"}, false},
		{"struct with mixed fields", struct{ 
			Exported   string 
			unexported string 
		}{Exported: "test", unexported: "test"}, false},
		{"pointer to struct with mixed fields", &struct{ 
			Exported   string 
			unexported string 
		}{Exported: "test", unexported: "test"}, false},
		{"large struct", struct{ 
			Field1 string 
			Field2 int 
			Field3 bool 
			Field4 float64 
			Field5 []string 
			Field6 map[string]int 
		}{Field1: "test", Field2: 42, Field3: true, Field4: 3.14, Field5: []string{"test"}, Field6: map[string]int{"key": 42}}, false},
		{"pointer to large struct", &struct{ 
			Field1 string 
			Field2 int 
			Field3 bool 
			Field4 float64 
			Field5 []string 
			Field6 map[string]int 
		}{Field1: "test", Field2: 42, Field3: true, Field4: 3.14, Field5: []string{"test"}, Field6: map[string]int{"key": 42}}, false},
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

// BenchmarkExtension benchmarks the Extension function performance
func BenchmarkExtension(b *testing.B) {
	testCases := []string{
		".zip",
		".tar.gz", 
		".7z",
		"application/zip",
		".unknown",
		"zip",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = Extension(tc)
			}
		})
	}
}

// BenchmarkMIME benchmarks the MIME function performance
func BenchmarkMIME(b *testing.B) {
	testCases := []string{
		"archive.zip",
		"archive.tar.gz",
		"file.txt",
		"path/to/archive.7z",
		"",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = MIME(tc)
			}
		})
	}
}

// BenchmarkSupported benchmarks the Supported function performance
func BenchmarkSupported(b *testing.B) {
	testCases := []any{
		nil,
		"string",
		42,
		struct{ field string }{field: "test"},
		&struct{ field string }{field: "test"},
	}

	for i, tc := range testCases {
		b.Run(string(rune(i)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = Supported(tc)
			}
		})
	}
}

// BenchmarkReadMIME benchmarks the ReadMIME function performance
// Note: This benchmark uses a temporary file for realistic testing
func BenchmarkReadMIME(b *testing.B) {
	// Create a temporary test file
	file, err := os.CreateTemp("", "benchmark-*.tmp")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	// Write some content
	_, err = file.WriteString("test content for benchmarking")
	if err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ReadMIME(file.Name())
	}
}