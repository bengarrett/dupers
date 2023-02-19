// © Ben Garrett https://github.com/bengarrett/dupers
package search_test

import (
	"testing"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/internal/task/internal/search"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
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
	err := search.Error(nil, true)
	assert.Nil(t, err)
	err = search.Error(database.ErrEmpty, true)
	assert.Nil(t, err)
	err = search.Error(bolt.ErrBucketNotFound, true)
	assert.Nil(t, err)
	err = search.Error(bolt.ErrDatabaseNotOpen, true)
	assert.NotNil(t, err)
}

func TestCompare(t *testing.T) {
	m, err := search.Compare(nil, nil, "", nil, true)
	assert.NotNil(t, err)
	assert.Nil(t, m)
	val := true
	f := cmd.Flags{
		Filename: &val,
		Exact:    &val,
	}
	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	m, err = search.Compare(db, &f, "", nil, true)
	assert.NotNil(t, err)
	assert.Nil(t, m)
}
