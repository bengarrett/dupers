// © Ben Garrett https://github.com/bengarrett/dupers
package search_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/cmd/task/search"
	"github.com/bengarrett/dupers/pkg/database"
	"github.com/nalgeon/be"
	"go.etcd.io/bbolt/errors"
)

func TestCmdErr(t *testing.T) {
	err := search.CmdErr(0, true)
	be.Err(t, err)
	err = search.CmdErr(-1, true)
	be.Err(t, err)
	err = search.CmdErr(1, true)
	be.Err(t, err)
	err = search.CmdErr(100, true)
	be.Err(t, err, nil)
}

func TestErr(t *testing.T) {
	err := search.Error(nil)
	be.Err(t, err, nil)
	err = search.Error(database.ErrEmpty)
	be.Err(t, err, nil)
	err = search.Error(errors.ErrBucketNotFound)
	be.Err(t, err)
	err = search.Error(errors.ErrDatabaseNotOpen)
	be.Err(t, err)
}

func TestCompare(t *testing.T) {
	m, err := search.Compare(nil, nil, "", nil)
	be.Err(t, err)
	be.Equal(t, m, nil)
	val := true
	f := cmd.Flags{
		Filename: &val,
		Exact:    &val,
	}
	db, path, err := mock.Database()
	be.Err(t, err, nil)
	defer db.Close()
	defer os.Remove(path)
	m, err = search.Compare(db, &f, "", nil)
	be.Err(t, err)
	be.Equal(t, m, nil)
}
