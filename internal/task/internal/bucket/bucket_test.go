// © Ben Garrett https://github.com/bengarrett/dupers
package bucket_test

import (
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/internal/task/internal/bucket"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/stretchr/testify/assert"
)

func TestExport(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Export(nil, false, args)
	assert.NotNil(t, err)
	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = bucket.Export(db, false, args)
	assert.NotNil(t, err)
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	args[1] = bucket1
	err = bucket.Export(db, false, args)
	assert.Nil(t, err)
}

func TestImport(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Import(nil, false, false, args)
	assert.NotNil(t, err)
	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = bucket.Import(db, false, false, args)
	assert.NotNil(t, err)
	args[1] = mock.CSV()
	err = bucket.Import(db, false, false, args)
	assert.Nil(t, err)
}

func TestList(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.List(nil, false, args)
	assert.NotNil(t, err)
	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = bucket.List(db, false, args)
	assert.NotNil(t, err)
	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	args[1] = bucket1
	err = bucket.List(db, false, args)
	assert.Nil(t, err)
}

func TestMove(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Move(nil, nil, false, args)
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = bucket.Move(db, nil, false, args)
	assert.NotNil(t, err)
	c := dupe.Config{}
	err = bucket.Move(db, &c, false, args)
	assert.NotNil(t, err)

	src, err := mock.Bucket(1)
	assert.Nil(t, err)
	dest, err := mock.Bucket(3)
	assert.Nil(t, err)
	args[0] = src
	args[1] = dest
	err = bucket.Move(db, &c, true, args)
	assert.Nil(t, err)
}

func TestRemove(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Remove(nil, false, false, args)
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = bucket.Remove(db, false, false, args)
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = bucket.Remove(db, false, false, args)
	assert.NotNil(t, err)

	args[1] = bucket1
	err = bucket.Remove(db, false, true, args)
	assert.Nil(t, err)

	args[1] = mock.NoSuchFile
	err = bucket.Remove(db, false, true, args)
	assert.NotNil(t, err)
}

func TestRescan(t *testing.T) {
	args := [2]string{"", ""}
	err := bucket.Rescan(nil, nil, false, args)
	assert.NotNil(t, err)
	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()
	err = bucket.Rescan(db, nil, false, args)
	assert.NotNil(t, err)

	c := dupe.Config{}
	err = bucket.Rescan(db, &c, false, args)
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = bucket.Rescan(db, &c, false, args)
	assert.NotNil(t, err)

	args[1] = bucket1
	err = bucket.Rescan(db, &c, false, args)
	assert.Nil(t, err)
}
