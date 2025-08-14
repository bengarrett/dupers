// © Ben Garrett https://github.com/bengarrett/dupers
package bucket_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/cmd/task/bucket"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/nalgeon/be"
)

func TestExport(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Export(nil, false, args)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = bucket.Export(db, false, args)
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	args[1] = bucket1
	err = bucket.Export(db, false, args)
	be.Err(t, err, nil)
}

func TestImport(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Import(nil, false, false, args)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = bucket.Import(db, false, false, args)
	be.Err(t, err)
	args[1] = mock.CSV(t)
	err = bucket.Import(db, false, false, args)
	be.Err(t, err, nil)
}

func TestList(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.List(nil, false, args)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = bucket.List(db, false, args)
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	args[1] = bucket1
	err = bucket.List(db, false, args)
	be.Err(t, err, nil)
}

func TestMove(t *testing.T) {
	err := bucket.Move(nil, nil, false, "", "")
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = bucket.Move(db, nil, false, "", "")
	be.Err(t, err)
	c := dupe.Config{}
	err = bucket.Move(db, &c, false, "", "")
	be.Err(t, err)
	src, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	dest, err := mock.Bucket(t, 3)
	be.Err(t, err, nil)
	err = bucket.Move(db, &c, true, src, dest)
	be.Err(t, err, nil)
}

func TestRemove(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Remove(nil, false, false, args)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = bucket.Remove(db, false, false, args)
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	err = bucket.Remove(db, false, false, args)
	be.Err(t, err)
	args[1] = bucket1
	err = bucket.Remove(db, false, true, args)
	be.Err(t, err, nil)
	args[1] = mock.NoSuchFile
	err = bucket.Remove(db, false, true, args)
	be.Err(t, err)
}

func TestRescan(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Rescan(nil, nil, false, args)
	be.Err(t, err)
	db, path := mock.Database(t)
	defer db.Close()
	defer os.Remove(path)
	err = bucket.Rescan(db, nil, false, args)
	be.Err(t, err)
	c := dupe.Config{}
	err = bucket.Rescan(db, &c, false, args)
	be.Err(t, err)
	bucket1, err := mock.Bucket(t, 1)
	be.Err(t, err, nil)
	err = bucket.Rescan(db, &c, false, args)
	be.Err(t, err)
	args[1] = bucket1
	err = bucket.Rescan(db, &c, false, args)
	be.Err(t, err, nil)
}
