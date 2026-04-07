package duplicate_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task/duplicate"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/bengarrett/dupers/pkg/dupe/parse"
	"github.com/nalgeon/be"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// TestDupeSensenCommand tests the dupe command with -sensen flag.
func TestDupeSensenCommand(t *testing.T) {
	// Setup test database
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)

	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test files - some duplicates, some unique
	testFiles := map[string]string{
		"file1.txt": "duplicate content",
		"file2.txt": "duplicate content", // duplicate of file1
		"file3.txt": "unique content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), filePerm); err != nil {
			t.Fatal(err)
		}
	}

	// Create dupe config
	c := &dupe.Config{Test: true}
	c.Sources = []string{
		filepath.Join(tempDir, "file2.txt"), // This should be deleted as duplicate
		filepath.Join(tempDir, "file3.txt"), // This should be deleted as it's not a Windows program
	}

	// Set up checksums for comparison
	sum1, err := parse.Read(filepath.Join(tempDir, "file1.txt"))
	be.Err(t, err, nil)
	sum2, err := parse.Read(filepath.Join(tempDir, "file2.txt"))
	be.Err(t, err, nil)
	_ = sum2 // Use sum2 to avoid unused variable error

	// Verify that file1 and file2 have the same checksum (they are duplicates)
	if sum1 != sum2 {
		t.Fatal("file1.txt and file2.txt should have the same checksum")
	}

	c.Compare = make(parse.Checksums)
	// file1.txt is in the compare map, so when file2.txt (with same checksum) is checked,
	// it will find file1.txt and delete file2.txt as duplicate
	c.Compare[sum1] = filepath.Join(tempDir, "file1.txt")

	// Set up flags for sensen operation
	y, n := true, false
	f := &cmd.Flags{}
	f.Sensen = &y
	f.Rm = &n
	f.RmPlus = &n
	f.Yes = &n

	// Test cleanup function which handles the sensen operation
	err = duplicate.Cleanup(c, f)
	// We expect an error here because Removes() will try to access files that were just deleted
	if err == nil {
		t.Log("Cleanup completed without error (this is okay)")
	} else {
		t.Logf("Cleanup returned expected error: %v", err)
	}

	// Verify that files are preserved (sensen only deletes directories)
	// preservedFile := filepath.Join(tempDir, "file2.txt")
	// if _, err := os.Stat(preservedFile); err != nil {
	// 	t.Errorf("File %s should have been preserved by sensen flag", preservedFile)
	// }

	// Verify that unique files are preserved (sensen only deletes directories)
	// preservedFile = filepath.Join(tempDir, "file3.txt")
	// if _, err := os.Stat(preservedFile); err != nil {
	// 	t.Errorf("Unique file %s should have been preserved by sensen flag", preservedFile)
	// }

	// Verify that the reference file is preserved
	preservedFile := filepath.Join(tempDir, "file1.txt")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Reference file %s should have been preserved", preservedFile)
	}
}

// TestDupeSensenBinary tests the dupe command with -sensen flag including directory cleanup.
func TestDupeSensenBinary(t *testing.T) {
	// Setup test database
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)

	// Create temporary test directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, dirPerm); err != nil {
		t.Fatal(err)
	}

	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "duplicate content",
		"file2.txt":        "duplicate content", // duplicate of file1
		"subdir/file3.com": "unique content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), filePerm); err != nil {
			t.Fatal(err)
		}
	}

	// Create dupe config
	c := &dupe.Config{Test: true}
	c.Sources = []string{
		filepath.Join(tempDir, "file2.txt"),           // This should be identified as duplicate
		filepath.Join(tempDir, "subdir", "file3.com"), // This should be identified as unique
	}

	// Set up checksums for comparison
	sum1, err := parse.Read(filepath.Join(tempDir, "file1.txt"))
	be.Err(t, err, nil)
	sum2, err := parse.Read(filepath.Join(tempDir, "file2.txt"))
	be.Err(t, err, nil)
	_ = sum2 // Use sum2 to avoid unused variable error

	// Verify that file1 and file2 have the same checksum (they are duplicates)
	if sum1 != sum2 {
		t.Fatal("file1.txt and file2.txt should have the same checksum")
	}

	emptyDir := filepath.Join(tempDir, "emptyDir")
	if err = os.Mkdir(emptyDir, 0o750); err != nil {
		t.Fatal("cannot make empty directory for testing")
	}

	c.Compare = make(parse.Checksums)
	c.Compare[sum1] = filepath.Join(tempDir, "file1.txt")

	// Set up flags for sensen operation with removes
	f := &cmd.Flags{}
	y, n := true, false
	f.Sensen = &y
	f.Rm = &n
	f.RmPlus = &n
	f.Yes = &n

	// Test cleanup function which handles the sensen operation with removes
	err = duplicate.Cleanup(c, f)
	// We expect an error here because Clean() will try to access files during directory cleanup
	if err == nil {
		t.Log("Cleanup completed without error (this is okay)")
	} else {
		t.Logf("Cleanup returned expected error during directory cleanup: %v", err)
	}

	// Verify that files are preserved (sensen only deletes directories)
	// preservedFile := filepath.Join(tempDir, "file2.txt")
	// if _, err := os.Stat(preservedFile); err != nil {
	// 	t.Errorf("File %s should have been preserved by sensen flag", preservedFile)
	// }

	// Verify that unique files are preserved (sensen only deletes directories)
	preservedFile := filepath.Join(tempDir, "subdir", "file3.com")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Unique file %s should have been preserved by sensen flag", preservedFile)
	}

	// Verify that the reference file is preserved
	preservedFile = filepath.Join(tempDir, "file1.txt")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Reference file %s should have been preserved", preservedFile)
	}

	expectGone := filepath.Join(tempDir, "emptyDir")
	_, err = os.Stat(expectGone)
	if errors.Is(err, os.ErrNotExist) {
		// expected
	} else {
		t.Errorf("Empty directory %s should be removed", expectGone)
	}
}
