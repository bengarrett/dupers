// © Ben Garrett https://github.com/bengarrett/dupers
package search_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task/search"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/stretchr/testify/assert"
	boltErr "go.etcd.io/bbolt/errors"
)

func TestCmdErr(t *testing.T) {
	err := search.CmdErr(0, true)
	assert.NotNil(t, err)
	err = search.CmdErr(-1, true)
	assert.NotNil(t, err)
	err = search.CmdErr(1, true)
	assert.NotNil(t, err)
	err = search.CmdErr(100, true)
	assert.Nil(t, err)
}

func TestErr(t *testing.T) {
	err := search.Error(nil)
	assert.Nil(t, err)
	err = search.Error(database.ErrEmpty)
	assert.Nil(t, err)
	err = search.Error(boltErr.ErrBucketNotFound)
	assert.NotNil(t, err)
	err = search.Error(boltErr.ErrDatabaseNotOpen)
	assert.NotNil(t, err)
}

func TestCompare(t *testing.T) {
	m, err := search.Compare(nil, nil, "", nil)
	assert.NotNil(t, err)
	assert.Nil(t, m)
	val := true
	f := cmd.Flags{
		Filename: &val,
		Exact:    &val,
	}
	db, path, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	defer os.Remove(path)
	m, err = search.Compare(db, &f, "", nil)
	assert.NotNil(t, err)
	assert.Nil(t, m)
}
