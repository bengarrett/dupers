// © Ben Garrett https://github.com/bengarrett/dupers

package archive_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bengarrett/dupers/pkg/dupe/internal/archive"
)

const (
	zipext = ".zip"
)

// TestFormatCache tests the format caching functionality.
func TestFormatCache(t *testing.T) {
	cache := archive.NewFormatCache()

	// Test cache miss (should identify format)
	format1, err := cache.GetFormat(context.Background(), "test.zip")
	if err != nil {
		t.Fatalf("Failed to get format: %v", err)
	}
	if format1 == nil {
		t.Fatal("Format should not be nil")
	}
	if format1.Extension() != zipext {
		t.Errorf("Expected .zip, got %s", format1.Extension())
	}

	// Test cache hit (should return cached format)
	format2, err := cache.GetFormat(context.Background(), "test.zip")
	if err != nil {
		t.Fatalf("Failed to get cached format: %v", err)
	}
	if format2 == nil {
		t.Fatal("Cached format should not be nil")
	}

	// Both should be the same object (cached)
	if format1 != format2 {
		t.Error("Format should be cached and reused")
	}

	// Test different format
	format3, err := cache.GetFormat(context.Background(), "test.tar.gz")
	if err != nil {
		t.Fatalf("Failed to get tar.gz format: %v", err)
	}
	if format3 == nil {
		t.Fatal("TAR format should not be nil")
	}
	if format3.Extension() != ".tar.gz" {
		t.Errorf("Expected .tar.gz, got %s", format3.Extension())
	}

	// Different formats should not be the same
	if format1 == format3 {
		t.Error("Different formats should not be the same object")
	}

	// Test cache clearing
	initialSize := cache.GetCacheSize()
	if initialSize != 2 {
		t.Errorf("Expected cache size 2 before clear, got %d", initialSize)
	}

	cache.ClearCache()
	finalSize := cache.GetCacheSize()
	if finalSize != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", finalSize)
	}

	// After clear, cache should be empty and we should be able to add new formats
	format4, err := cache.GetFormat(context.Background(), "test.zip")
	if err != nil {
		t.Fatalf("Failed to get format after clear: %v", err)
	}
	if format4 == nil || format4.Extension() != zipext {
		t.Error("Expected valid ZIP format after clear")
	}

	// Cache should have one item again
	if cache.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1 after re-add, got %d", cache.GetCacheSize())
	}
}

// TestOptimizedExtractor tests the optimized extractor.
func TestOptimizedExtractor(t *testing.T) {
	extractor := archive.NewOptimizedExtractor()

	// Test that it was created successfully
	if extractor == nil {
		t.Fatal("Failed to create OptimizedExtractor")
	}

	// Test that it has a working cache
	if extractor.Cache == nil {
		t.Fatal("OptimizedExtractor should have a cache")
	}

	// Test cache functionality through the extractor
	cache := extractor.Cache
	format, err := cache.GetFormat(context.Background(), "test.zip")
	if err != nil {
		t.Fatalf("Failed to get format: %v", err)
	}
	if format == nil || format.Extension() != zipext {
		t.Error("Expected valid ZIP format")
	}

	// Test cache reuse
	format2, err := cache.GetFormat(context.Background(), "test.zip")
	if err != nil {
		t.Fatalf("Failed to get cached format: %v", err)
	}
	if format != format2 {
		t.Error("Format should be cached and reused")
	}
}

// TestBatchExtractor tests the batch extractor.
func TestBatchExtractor(t *testing.T) {
	extractor := archive.NewBatchExtractor()

	// Test that it was created successfully
	if extractor == nil {
		t.Fatal("Failed to create BatchExtractor")
	}

	// Test that it has a working cache
	if extractor.Cache == nil {
		t.Fatal("BatchExtractor should have a cache")
	}

	// Test cache functionality
	cache := extractor.Cache
	format, err := cache.GetFormat(context.Background(), "test.tar.gz")
	if err != nil {
		t.Fatalf("Failed to get format: %v", err)
	}
	if format == nil || format.Extension() != ".tar.gz" {
		t.Error("Expected valid TAR.GZ format")
	}

	// Test cache size
	initialSize := cache.GetCacheSize()
	if initialSize != 1 {
		t.Errorf("Expected cache size 1, got %d", initialSize)
	}

	// Test cache clearing
	cache.ClearCache()
	if cache.GetCacheSize() != 0 {
		t.Error("Cache should be empty after clear")
	}
}

// BenchmarkFormatCache benchmarks cache performance.
func BenchmarkFormatCache(b *testing.B) {
	cache := archive.NewFormatCache()

	b.Run("CacheMiss", func(b *testing.B) {
		for i := range b.N {
			// Use different filename each time to avoid cache hits
			_, _ = cache.GetFormat(context.Background(), fmt.Sprintf("test%d.zip", i))
		}
	})

	b.Run("CacheHit", func(b *testing.B) {
		// Prime the cache
		_, _ = cache.GetFormat(context.Background(), "cached.zip")

		b.ResetTimer()
		for range b.N {
			_, _ = cache.GetFormat(context.Background(), "cached.zip")
		}
	})
}

// BenchmarkBufferPool benchmarks buffer pool performance.
func BenchmarkBufferPool(b *testing.B) {
	extractor := archive.NewOptimizedExtractor()

	b.Run("WithPool", func(b *testing.B) {
		for range b.N {
			pool, ok := extractor.BufferPool.Get().(*[]byte)
			if !ok {
				b.Fatal("Buffer pool returned unexpected type")
			}
			buf := *pool
			_ = buf[0] // Use the buffer
			extractor.BufferPool.Put(pool)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for range b.N {
			buf := make([]byte, 1024*1024)
			_ = buf[0] // Use the buffer
		}
	})
}
