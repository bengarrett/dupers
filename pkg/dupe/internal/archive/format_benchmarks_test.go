// © Ben Garrett https://github.com/bengarrett/dupers

package archive_test

import (
	"context"
	"os"
	"testing"

	"github.com/bengarrett/dupers/pkg/dupe/internal/archive"
	"github.com/mholt/archives"
)

// BenchmarkFormatIdentification benchmarks format identification for different archive types
func BenchmarkFormatIdentification(b *testing.B) {
	testFiles := map[string]string{
		"ZIP":  "../../../../testdata/randomfiles.zip",
		"TAR":  "../../../../testdata/randomfiles.tar.xz",
		"7Z":   "../../../../testdata/randomfiles.7z",
	}

	ctx := context.Background()

	for name, file := range testFiles {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := archives.Identify(ctx, file, nil)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

// BenchmarkOldVsNewIdentification compares old vs new API identification performance
func BenchmarkOldVsNewIdentification(b *testing.B) {
	testFiles := []string{
		"../../../../testdata/randomfiles.zip",
		"../../../../testdata/randomfiles.tar.xz",
		"../../../../testdata/randomfiles.7z",
	}

	for _, file := range testFiles {
		b.Run(file, func(b *testing.B) {
			// Old API benchmark
			b.Run("old", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := archive.ReadMIME(file)
					if err != nil {
						b.Error(err)
					}
				}
			})

			// New API benchmark
			b.Run("new", func(b *testing.B) {
				ctx := context.Background()
				for i := 0; i < b.N; i++ {
					_, _, err := archives.Identify(ctx, file, nil)
					if err != nil {
						b.Error(err)
					}
				}
			})
		})
	}
}

// BenchmarkExtractionPerformance benchmarks file extraction performance
func BenchmarkExtractionPerformance(b *testing.B) {
	testFiles := map[string]string{
		"ZIP":  "../../../../testdata/randomfiles.zip",
		"TAR":  "../../../../testdata/randomfiles.tar.xz",
		"7Z":   "../../../../testdata/randomfiles.7z",
	}

	ctx := context.Background()

	for name, file := range testFiles {
		b.Run(name, func(b *testing.B) {
			// Open the file
			f, err := os.Open(file)
			if err != nil {
				b.Skipf("Test file not found: %s", file)
			}
			defer f.Close()

			// Identify format
			format, _, err := archives.Identify(ctx, file, f)
			if err != nil {
				b.Skipf("Failed to identify format: %v", err)
			}

			_, ok := format.(archives.Extractor)
			if !ok {
				b.Skipf("Format does not support extraction")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Reset file pointer
				if _, err := f.Seek(0, 0); err != nil {
					b.Error(err)
					break
				}

				// Re-identify format
				format, reader, err := archives.Identify(ctx, file, f)
				if err != nil {
					b.Error(err)
					break
				}

				extractor, ok := format.(archives.Extractor)
				if !ok {
					b.Error("Format does not support extraction")
					break
				}

				// Extract files
				err = extractor.Extract(ctx, reader, func(ctx context.Context, fileInfo archives.FileInfo) error {
					if !fileInfo.IsDir() {
						// Open and read file to simulate real usage
						file, err := fileInfo.Open()
						if err != nil {
							return err
						}
						defer file.Close()
						
						buf := make([]byte, 1024)
						_, _ = file.Read(buf)
					}
					return nil
				})
				
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	testFiles := map[string]string{
		"ZIP":  "../../../../testdata/randomfiles.zip",
		"TAR":  "../../../../testdata/randomfiles.tar.xz",
		"7Z":   "../../../../testdata/randomfiles.7z",
	}

	ctx := context.Background()

	for name, file := range testFiles {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, _, err := archives.Identify(ctx, file, nil)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}