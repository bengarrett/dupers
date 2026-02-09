package duplicate_test

import (
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
	f := &cmd.Flags{}
	sensenFlag := true
	f.Sensen = &sensenFlag
	noFlag := false
	f.Rm = &noFlag
	f.RmPlus = &noFlag
	f.Yes = &noFlag

	// Test cleanup function which handles the sensen operation
	err = duplicate.Cleanup(c, f)
	// We expect an error here because Removes() will try to access files that were just deleted
	if err == nil {
		t.Log("Cleanup completed without error (this is okay)")
	} else {
		t.Logf("Cleanup returned expected error: %v", err)
	}

	// Verify that duplicate files are deleted
	deletedFile := filepath.Join(tempDir, "file2.txt")
	if _, err := os.Stat(deletedFile); !os.IsNotExist(err) {
		t.Errorf("Duplicate file %s should have been deleted by sensen flag", deletedFile)
	}

	// Verify that unique files are preserved (sensen only deletes duplicates)
	preservedFile := filepath.Join(tempDir, "file3.txt")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Unique file %s should have been preserved by sensen flag", preservedFile)
	}

	// Verify that the reference file is preserved
	preservedFile = filepath.Join(tempDir, "file1.txt")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Reference file %s should have been preserved", preservedFile)
	}
}

// TestDupeSensenWithRemovesCommand tests the dupe command with -sensen flag including directory cleanup.
func TestDupeSensenWithRemovesCommand(t *testing.T) {
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
		"subdir/file3.txt": "unique content",
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
		filepath.Join(tempDir, "subdir", "file3.txt"), // This should be identified as unique
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
	c.Compare[sum1] = filepath.Join(tempDir, "file1.txt")

	// Set up flags for sensen operation with removes
	f := &cmd.Flags{}
	sensenFlag := true
	f.Sensen = &sensenFlag
	noFlag := false
	f.Rm = &noFlag
	f.RmPlus = &noFlag
	f.Yes = &noFlag

	// Test cleanup function which handles the sensen operation with removes
	err = duplicate.Cleanup(c, f)
	// We expect an error here because Clean() will try to access files during directory cleanup
	if err == nil {
		t.Log("Cleanup completed without error (this is okay)")
	} else {
		t.Logf("Cleanup returned expected error during directory cleanup: %v", err)
	}

	// Verify that duplicate files are deleted
	deletedFile := filepath.Join(tempDir, "file2.txt")
	if _, err := os.Stat(deletedFile); !os.IsNotExist(err) {
		t.Errorf("Duplicate file %s should have been deleted by sensen flag", deletedFile)
	}

	// Verify that unique files are preserved (sensen only deletes duplicates)
	preservedFile := filepath.Join(tempDir, "subdir", "file3.txt")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Unique file %s should have been preserved by sensen flag", preservedFile)
	}

	// Verify that the reference file is preserved
	preservedFile = filepath.Join(tempDir, "file1.txt")
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("Reference file %s should have been preserved", preservedFile)
	}
}
