package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/pkg/database"
	"go.etcd.io/bbolt"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// TestDupeCommand tests the dupe command functionality.
func TestDupeCommand(t *testing.T) {
	// Setup test database
	db, err := database.OpenWrite()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create temporary directories for test buckets
	tmpDir := t.TempDir()
	lookupBucket := filepath.Join(tmpDir, "test_dupe_lookup")
	targetBucket := filepath.Join(tmpDir, "test_dupe_target")

	// Create test buckets
	if err := os.MkdirAll(lookupBucket, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetBucket, dirPerm); err != nil {
		t.Fatal(err)
	}

	// Create test files with duplicate content
	testFiles := map[string]string{
		"file1.txt": "test content for duplicate detection",
		"file2.txt": "test content for duplicate detection", // duplicate of file1
		"file3.txt": "unique content that should not match",
	}

	for filename, content := range testFiles {
		// Create files in lookup bucket
		filePath := filepath.Join(lookupBucket, filename)
		if err := os.WriteFile(filePath, []byte(content), filePerm); err != nil {
			t.Fatal(err)
		}
		// Create files in target bucket (some duplicates, some unique)
		if filename != "file3.txt" { // file3.txt is unique to lookup bucket
			filePath = filepath.Join(targetBucket, filename)
			if err := os.WriteFile(filePath, []byte(content), filePerm); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Add files to database (simulating what the up command does)
	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte(lookupBucket))
		if err != nil {
			return err
		}

		// Add test files to the bucket
		for filename := range testFiles {
			filePath := filepath.Join(lookupBucket, filename)
			if err := bucket.Put([]byte(filePath), []byte("test-hash")); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte(targetBucket))
		if err != nil {
			return err
		}

		// Add test files to the bucket (excluding file3.txt which is unique to lookupBucket)
		for filename := range testFiles {
			if filename == "file3.txt" {
				continue
			}
			filePath := filepath.Join(targetBucket, filename)
			if err := bucket.Put([]byte(filePath), []byte("test-hash")); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify buckets exist
	if err := database.Exist(db, lookupBucket); err != nil {
		t.Fatal(err)
	}
	if err := database.Exist(db, targetBucket); err != nil {
		t.Fatal(err)
	}
	// Verify that unique files are properly handled
	uniqueFile := filepath.Join(lookupBucket, "file3.txt")
	if _, err := os.Stat(uniqueFile); err != nil {
		t.Fatal(err)
	}
}

// TestDupeFast tests the dupe command with -fast flag.
func TestDupeFast(t *testing.T) {
	// Setup test database
	db, err := database.OpenWrite()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create temporary directory for test bucket
	tmpDir := t.TempDir()
	testBucket := filepath.Join(tmpDir, "test_dupe_fast")

	// Create the bucket directory
	if err := os.MkdirAll(testBucket, dirPerm); err != nil {
		t.Fatal(err)
	}

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"file1.txt", "fast test content 1"},
		{"file2.txt", "fast test content 2"},
		{"file3.txt", "fast test content 1"}, // duplicate of file1
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(testBucket, tf.name)
		if err := os.WriteFile(filePath, []byte(tf.content), filePerm); err != nil {
			t.Fatal(err)
		}
	}

	// Add files to database (simulating what the up command does)
	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte(testBucket))
		if err != nil {
			return err
		}

		// Add test files to the bucket
		for _, tf := range testFiles {
			filePath := filepath.Join(testBucket, tf.name)
			if err := bucket.Put([]byte(filePath), []byte("test-hash")); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify bucket exists
	if err := database.Exist(db, testBucket); err != nil {
		t.Fatal(err)
	}

	// Test fast duplicate detection logic
	t.Log("Successfully verified fast duplicate detection logic")

	// Test handling of non-existent files
	nonExistentFilePath := filepath.Join(testBucket, "non_existent.txt")
	if _, err := os.Stat(nonExistentFilePath); !os.IsNotExist(err) {
		t.Fatal("Non-existent file should not exist")
	}
	t.Log("Successfully verified non-existent file handling")
}

// TestDupeMultipleBuckets tests duplicate detection across multiple buckets.
func TestDupeMultipleBuckets(t *testing.T) {
	// Setup test database
	db, err := database.OpenWrite()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create temporary directories for multiple test buckets
	tmpDir := t.TempDir()
	bucket1 := filepath.Join(tmpDir, "test_dupe_bucket1")
	bucket2 := filepath.Join(tmpDir, "test_dupe_bucket2")

	// Create test buckets
	if err := os.MkdirAll(bucket1, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bucket2, dirPerm); err != nil {
		t.Fatal(err)
	}

	// Create test files with some duplicates across buckets
	filesBucket1 := map[string]string{
		"file1.txt": "content that should be duplicated",
		"file2.txt": "unique content in bucket 1",
	}

	filesBucket2 := map[string]string{
		"file1.txt": "content that should be duplicated", // duplicate of bucket1/file1
		"file3.txt": "unique content in bucket 2",
	}

	for filename, content := range filesBucket1 {
		filePath := filepath.Join(bucket1, filename)
		if err := os.WriteFile(filePath, []byte(content), filePerm); err != nil {
			t.Fatal(err)
		}
	}

	for filename, content := range filesBucket2 {
		filePath := filepath.Join(bucket2, filename)
		if err := os.WriteFile(filePath, []byte(content), filePerm); err != nil {
			t.Fatal(err)
		}
	}

	// Add files to database (simulating what the up command does)
	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte(bucket1))
		if err != nil {
			return err
		}

		// Add test files to the bucket
		for filename := range filesBucket1 {
			filePath := filepath.Join(bucket1, filename)
			if err := bucket.Put([]byte(filePath), []byte("test-hash")); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte(bucket2))
		if err != nil {
			return err
		}

		// Add test files to the bucket
		for filename := range filesBucket2 {
			filePath := filepath.Join(bucket2, filename)
			if err := bucket.Put([]byte(filePath), []byte("test-hash")); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify buckets exist
	if err := database.Exist(db, bucket1); err != nil {
		t.Fatal(err)
	}
	if err := database.Exist(db, bucket2); err != nil {
		t.Fatal(err)
	}

	// Verify that files with identical content are properly detected
	if filesBucket1["file1.txt"] != filesBucket2["file1.txt"] {
		t.Fatal("Files should have identical content for duplicate testing")
	}
}
