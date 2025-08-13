// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task/duplicate"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/nalgeon/be"
)

func TestCleanup(t *testing.T) {
	err := duplicate.Cleanup(nil, nil)
	be.Err(t, err)
	c := dupe.Config{}
	f := cmd.Flags{}
	err = duplicate.Cleanup(&c, nil)
	be.Err(t, err)
	err = duplicate.Cleanup(nil, &f)
	be.Err(t, err)
	err = duplicate.Cleanup(&c, &f)
	be.Err(t, err)
	no, yes := false, true
	f.Sensen = &no
	f.Rm = &no
	f.RmPlus = &no
	err = duplicate.Cleanup(&c, &f)
	be.Err(t, err)
	f.Sensen = &yes
	f.Yes = &no
	err = duplicate.Cleanup(&c, &f)
	be.Err(t, err)
}

func TestWalkScanSave(t *testing.T) {
	err := duplicate.WalkScanSave(nil, nil, nil)
	be.Err(t, err)
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	err = duplicate.WalkScanSave(db, nil, nil)
	be.Err(t, err)
	c := dupe.Config{Test: true}
	err = duplicate.WalkScanSave(db, &c, nil)
	be.Err(t, err)
	f := cmd.Flags{}
	err = duplicate.WalkScanSave(db, &c, &f)
	be.Err(t, err)
	no, yes := false, true
	f.Lookup = &no
	err = duplicate.WalkScanSave(db, &c, &f)
	be.Err(t, err, nil)
	f.Lookup = &yes
	err = duplicate.WalkScanSave(db, &c, &f)
	be.Err(t, err, nil)
}

func TestLookup(t *testing.T) {
	err := duplicate.Lookup(nil, nil)
	be.Err(t, err)
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	err = duplicate.Lookup(db, nil)
	be.Err(t, err)
	c := dupe.Config{Test: true}
	err = duplicate.Lookup(db, &c)
	be.Err(t, err, nil)
	err = duplicate.Lookup(db, &c)
	be.Err(t, err, nil)
}
