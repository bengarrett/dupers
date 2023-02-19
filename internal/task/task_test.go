// © Ben Garrett https://github.com/bengarrett/dupers
package task_test

import (
	"flag"
	"testing"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/internal/task"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/stretchr/testify/assert"
)

func TestHelps(t *testing.T) {
	assert.NotEqual(t, "", task.Help())
	assert.NotEqual(t, "", task.HelpDatabase())
	assert.NotEqual(t, "", task.HelpDupe())
	assert.NotEqual(t, "", task.HelpSearch())

	s, err := task.Debug(nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "", s)
	a := cmd.Aliases{}
	s, err = task.Debug(&a, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "", s)
	f := cmd.Flags{}
	s, err = task.Debug(nil, &f)
	assert.NotNil(t, err)
	assert.Equal(t, "", s)
}

func TestWalkScan(t *testing.T) {
	err := task.WalkScan(nil, nil, nil, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = task.WalkScan(db, nil, nil, "")
	assert.NotNil(t, err)

	c := dupe.Config{}
	err = task.WalkScan(db, &c, nil, "")
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	bucket2, err := mock.Bucket(2)
	assert.Nil(t, err)
	args := []string{bucket2}
	err = task.WalkScan(db, &c, nil, args...)
	assert.NotNil(t, err)

	f := cmd.Flags{}
	err = c.SetSource(bucket1)
	assert.Nil(t, err)
	err = task.WalkScan(db, &c, &f, args...)
	assert.NotNil(t, err)
	lookup := false
	f.Lookup = &lookup
	err = task.WalkScan(db, &c, &f, args...)
	assert.NotNil(t, err)
}
func TestWalkCheck(t *testing.T) {
	err := task.WalkCheck(nil, nil, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = task.WalkCheck(db, nil, "")
	assert.NotNil(t, err)
	c := dupe.Config{}
	err = task.WalkCheck(db, &c, "")
	assert.NotNil(t, err)
	args := []string{"placeholder 1", "placeholder 2"}
	err = task.WalkCheck(db, &c, args...)
	assert.Nil(t, err)
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	args = []string{"placeholder 1", "placeholder 2", bucket1}
	err = task.WalkCheck(db, &c, args...)
	assert.NotNil(t, err)
}

func TestSearch(t *testing.T) {
	err := task.Search(nil, nil, true, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = task.Search(db, nil, true, "")
	assert.NotNil(t, err)

	f := cmd.Flags{}
	err = task.Search(db, &f, true, "")
	assert.NotNil(t, err)

	// Usage:
	// dupers [options] search <search expression> [optional, buckets to search]
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	args := []string{"hello", bucket1}
	err = task.Search(db, &f, true, args...)
	assert.NotNil(t, err)

	f.Filename = flag.Bool("a", false, "")
	f.Exact = flag.Bool("b", false, "")
	f.Quiet = flag.Bool("c", false, "")
	err = task.Search(db, &f, true, args...)
	assert.Nil(t, err)
}

func TestDupe(t *testing.T) {
	err := task.Dupe(nil, nil, nil, true, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = task.Dupe(db, nil, nil, true, "")
	assert.NotNil(t, err)

	c := dupe.Config{}
	err = task.Dupe(db, &c, nil, true, "")
	assert.NotNil(t, err)
	f := cmd.Flags{}
	err = task.Dupe(db, &c, &f, true, "")
	assert.NotNil(t, err)

	// there's no need to run further tests
	// as they can be done using the Taskfile.yaml
}

func TestDatabase(t *testing.T) {
	err := task.Database(nil, nil, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = task.Database(db, nil, "")
	assert.NotNil(t, err)

	args := []string{""}
	err = task.Database(db, nil, args...)
	assert.NotNil(t, err)

	args = []string{"qwerty", "asdfgh"}
	err = task.Database(db, nil, args...)
	assert.NotNil(t, err)

	c := dupe.Config{
		Test: true,
	}
	err = task.Database(db, &c, args...)
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	args = []string{"ls", bucket1}
	err = task.Database(db, &c, args...)
	assert.Nil(t, err)
}

func TestCleanupDB(t *testing.T) {
	err := task.CleanupDB(nil, nil)
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = task.CleanupDB(db, nil)
	assert.NotNil(t, err)
	c := dupe.Config{}
	err = task.CleanupDB(db, &c)
	assert.Nil(t, err)
}

func TestCheckDupePaths(t *testing.T) {
	err := task.CheckDupePaths(nil)
	assert.NotNil(t, err)
	c := dupe.Config{}
	err = task.CheckDupePaths(&c)
	assert.NotNil(t, err)
}

// 	type args struct {
// 		c   *dupe.Config
// 		cmd string
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{"empty", args{}, true},
// 		{"invalid cmd", args{cmd: "xyz"}, true},
// 		{"backup", args{cmd: "backup"}, false},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := task.Database(nil, tt.args.c, tt.args.cmd); (err != nil) != tt.wantErr {
// 				t.Errorf("Database() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }
