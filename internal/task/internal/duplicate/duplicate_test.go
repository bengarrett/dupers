// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/internal/task/internal/duplicate"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/stretchr/testify/assert"
)

func TestCleanup(t *testing.T) {
	err := duplicate.Cleanup(nil, nil)
	assert.NotNil(t, err)
	c := dupe.Config{}
	f := cmd.Flags{}
	err = duplicate.Cleanup(&c, nil)
	assert.NotNil(t, err)
	err = duplicate.Cleanup(nil, &f)
	assert.NotNil(t, err)
	err = duplicate.Cleanup(&c, &f)
	assert.NotNil(t, err)
	no, yes := false, true
	f.Sensen = &no
	f.Rm = &no
	f.RmPlus = &no
	err = duplicate.Cleanup(&c, &f)
	assert.NotNil(t, err)
	f.Sensen = &yes
	f.Yes = &no
	err = duplicate.Cleanup(&c, &f)
	assert.NotNil(t, err)
}

func TestWalkScanSave(t *testing.T) {
	err := duplicate.WalkScanSave(nil, nil, nil)
	assert.NotNil(t, err)
	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)
	err = duplicate.WalkScanSave(db, nil, nil)
	assert.NotNil(t, err)
	c := dupe.Config{Test: true}
	err = duplicate.WalkScanSave(db, &c, nil)
	assert.NotNil(t, err)
	f := cmd.Flags{}
	err = duplicate.WalkScanSave(db, &c, &f)
	assert.NotNil(t, err)
	no, yes := false, true
	f.Lookup = &no
	err = duplicate.WalkScanSave(db, &c, &f)
	assert.Nil(t, err)
	f.Lookup = &yes
	err = duplicate.WalkScanSave(db, &c, &f)
	assert.Nil(t, err)
}

func TestLookup(t *testing.T) {
	err := duplicate.Lookup(nil, nil)
	assert.NotNil(t, err)
	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)
	err = duplicate.Lookup(db, nil)
	assert.NotNil(t, err)
	c := dupe.Config{Test: true}
	err = duplicate.Lookup(db, &c)
	assert.Nil(t, err)
	err = duplicate.Lookup(db, &c)
	assert.Nil(t, err)
}
