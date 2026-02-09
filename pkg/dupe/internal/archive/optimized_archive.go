// © Ben Garrett https://github.com/bengarrett/dupers

package archive

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/mholt/archives"
)

var (
	ErrUnsupportedFormat = errors.New("format does not support extraction")
	ErrNotFound          = errors.New("archive not found in cache")
)

// FormatCache caches format identification results to avoid repeated work.
type FormatCache struct {
	Mu    sync.RWMutex
	Cache map[string]archives.Format
}

// NewFormatCache creates a new format cache.
func NewFormatCache() *FormatCache {
	return &FormatCache{
		Cache: make(map[string]archives.Format),
	}
}

// GetFormat returns a cached format or identifies a new one.
func (fc *FormatCache) GetFormat(ctx context.Context, filename string) (archives.Format, error) { //nolint:ireturn
	fc.Mu.RLock()
	if format, ok := fc.Cache[filename]; ok {
		fc.Mu.RUnlock()
		return format, nil
	}
	fc.Mu.RUnlock()

	// Identify new format
	format, _, err := archives.Identify(ctx, filename, nil)
	if err != nil {
		return nil, err
	}

	// Cache the result
	fc.Mu.Lock()
	fc.Cache[filename] = format
	fc.Mu.Unlock()

	return format, nil
}

// ClearCache clears the format cache.
func (fc *FormatCache) ClearCache() {
	fc.Mu.Lock()
	fc.Cache = make(map[string]archives.Format)
	fc.Mu.Unlock()
}

// GetCacheSize returns the number of cached formats.
func (fc *FormatCache) GetCacheSize() int {
	fc.Mu.RLock()
	defer fc.Mu.RUnlock()
	return len(fc.Cache)
}

// OptimizedExtractor provides optimized extraction with caching and reuse.
type OptimizedExtractor struct {
	Cache      *FormatCache
	BufferPool sync.Pool
}

// NewOptimizedExtractor creates a new optimized extractor.
func NewOptimizedExtractor() *OptimizedExtractor {
	const oneMB = 1024 * 1024
	return &OptimizedExtractor{
		Cache: NewFormatCache(),
		BufferPool: sync.Pool{
			New: func() any {
				buf := make([]byte, oneMB)
				return &buf
			},
		},
	}
}

// ExtractWithCache extracts files using cached format identification.
func (oe *OptimizedExtractor) ExtractWithCache(filename string, reader io.Reader,
	handleFile func(fileInfo archives.FileInfo) error,
) error {
	// Get cached format
	format, err := oe.Cache.GetFormat(context.Background(), filename)
	if err != nil {
		return err
	}

	// Check if format supports extraction
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return ErrUnsupportedFormat
	}

	// Use buffered extraction with pool
	return extractor.Extract(context.Background(), reader, func(ctx context.Context, fileInfo archives.FileInfo) error {
		if fileInfo.IsDir() {
			return nil
		}

		// Use buffer pool for reading
		bufPtr, ok := oe.BufferPool.Get().(*[]byte)
		if !ok {
			buf := make([]byte, 0)
			bufPtr = &buf
		}
		buf := *bufPtr
		defer oe.BufferPool.Put(&buf)

		file, err := fileInfo.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("Could not close file: %v", err)
			}
		}()

		// For checksum calculation (like dupers does)
		h := sha256.New()
		if _, err := io.CopyBuffer(h, file, buf); err != nil {
			return err
		}
		_ = h.Sum(nil) // Calculate checksum

		return handleFile(fileInfo)
	})
}

// BatchExtractor handles batch processing of multiple files from archives.
type BatchExtractor struct {
	Cache     *FormatCache
	FileCache map[string][]archives.FileInfo
	FileMutex sync.Mutex
}

// NewBatchExtractor creates a new batch extractor.
func NewBatchExtractor() *BatchExtractor {
	return &BatchExtractor{
		Cache:     NewFormatCache(),
		FileCache: make(map[string][]archives.FileInfo),
	}
}

// CacheArchiveFiles caches all files from an archive for batch processing.
func (be *BatchExtractor) CacheArchiveFiles(filename string, reader io.Reader) error {
	format, err := be.Cache.GetFormat(context.Background(), filename)
	if err != nil {
		return err
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return ErrUnsupportedFormat
	}

	var files []archives.FileInfo
	err = extractor.Extract(context.Background(), reader, func(ctx context.Context, fileInfo archives.FileInfo) error {
		if !fileInfo.IsDir() {
			files = append(files, fileInfo)
		}
		return nil
	})
	if err != nil {
		return err
	}

	be.FileMutex.Lock()
	be.FileCache[filename] = files
	be.FileMutex.Unlock()

	return nil
}

// ProcessCachedFiles processes all cached files from an archive.
func (be *BatchExtractor) ProcessCachedFiles(filename string,
	processFunc func(fileInfo archives.FileInfo) error,
) error {
	be.FileMutex.Lock()
	files, ok := be.FileCache[filename]
	be.FileMutex.Unlock()

	if !ok {
		return ErrNotFound
	}

	for _, fileInfo := range files {
		if err := processFunc(fileInfo); err != nil {
			return err
		}
	}

	return nil
}

// ClearFileCache clears the file cache.
func (be *BatchExtractor) ClearFileCache() {
	be.FileMutex.Lock()
	be.FileCache = make(map[string][]archives.FileInfo)
	be.FileMutex.Unlock()
}
