// © Ben Garrett https://github.com/bengarrett/dupers
package task_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/nalgeon/be"
)

func TestHelps(t *testing.T) {
	be.True(t, task.Help() != "")
	be.True(t, task.HelpDatabase() != "")
	be.True(t, task.HelpDupe() != "")
	be.True(t, task.HelpSearch() != "")
	s, err := task.Debug(nil, nil)
	be.Err(t, err)
	be.Equal(t, s, "")
	a := cmd.Aliases{}
	s, err = task.Debug(&a, nil)
	be.Err(t, err)
	be.Equal(t, s, "")
	f := cmd.Flags{}
	s, err = task.Debug(nil, &f)
	be.Err(t, err)
	be.Equal(t, s, "")
}

func TestWalkScan(t *testing.T) {
	err := task.WalkScan(nil, nil, nil, "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = task.WalkScan(db, nil, nil, "")
	be.Err(t, err)
	c := dupe.Config{}
	err = task.WalkScan(db, &c, nil, "")
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	bucket2, err := mock.Bucket(t, 2)
	be.Err(t, err, nil)
	args := []string{bucket2}
	err = task.WalkScan(db, &c, nil, args...)
	be.Err(t, err)
	f := cmd.Flags{}
	err = c.SetSource(bucket1)
	be.Err(t, err, nil)
	err = task.WalkScan(db, &c, &f, args...)
	be.Err(t, err)
	lookup := false
	f.Lookup = &lookup
	err = task.WalkScan(db, &c, &f, args...)
	be.Err(t, err)
}

func TestSetStat(t *testing.T) {
	err := task.SetStat(nil, nil, "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = task.SetStat(db, nil, "")
	be.Err(t, err)
	c := dupe.Config{}
	err = task.SetStat(db, &c, "")
	be.Err(t, err)
	args := []string{"placeholder 1", "placeholder 2"}
	err = task.SetStat(db, &c, args...)
	be.Err(t, err, nil)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	args = []string{"placeholder 1", "placeholder 2", bucket1}
	err = task.SetStat(db, &c, args...)
	be.Err(t, err)
}

func TestSearch(t *testing.T) {
	err := task.Search(nil, nil, true, "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = task.Search(db, nil, true, "")
	be.Err(t, err)
	f := cmd.Flags{}
	err = task.Search(db, &f, true, "")
	be.Err(t, err)
	// Usage:
	// dupers [options] search <search expression> [optional, buckets to search]
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	args := []string{"hello", bucket1}
	err = task.Search(db, &f, true, args...)
	be.Err(t, err)
	no := false
	f.Filename = &no
	f.Exact = &no
	f.Quiet = &no
	err = task.Search(db, &f, true, args...)
	be.Err(t, err, nil)
}

func TestDupe(t *testing.T) {
	err := task.Dupe(nil, nil, nil, "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = task.Dupe(db, nil, nil, "")
	be.Err(t, err)
	c := dupe.Config{}
	err = task.Dupe(db, &c, nil, "")
	be.Err(t, err)
	f := cmd.Flags{}
	err = task.Dupe(db, &c, &f, "")
	be.Err(t, err)
	// there's no need to run further tests
	// as they can be done using the Taskfile.yaml
}

func TestDatabase(t *testing.T) {
	err := task.Database(nil, nil, "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = task.Database(db, nil, "")
	be.Err(t, err)
	args := []string{""}
	err = task.Database(db, nil, args...)
	be.Err(t, err)
	args = []string{"qwerty", "asdfgh"}
	err = task.Database(db, nil, args...)
	be.Err(t, err)
	c := dupe.Config{
		Test: true,
	}
	err = task.Database(db, &c, args...)
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	args = []string{"ls", bucket1}
	err = task.Database(db, &c, args...)
	be.Err(t, err, nil)
}

func TestCleanupDB(t *testing.T) {
	err := task.CleanupDB(nil, nil)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = task.CleanupDB(db, nil)
	be.Err(t, err)
	c := dupe.Config{}
	err = task.CleanupDB(db, &c)
	be.Err(t, err, nil)
}

func TestStatSource(t *testing.T) {
	err := task.StatSource(nil)
	be.Err(t, err)
	c := dupe.Config{}
	err = task.StatSource(&c)
	be.Err(t, err)
}
