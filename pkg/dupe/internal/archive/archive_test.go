// © Ben Garrett https://github.com/bengarrett/dupers
package archive_test

import (
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
	"github.com/bengarrett/dupers/pkg/dupe"
	"github.com/bengarrett/dupers/pkg/dupe/internal/archive"
	"github.com/bengarrett/dupers/pkg/dupe/internal/parse"
	"github.com/stretchr/testify/assert"
)

func TestExtension(t *testing.T) {
	s := archive.Extension("")
	assert.Equal(t, "", s)

	s = archive.Extension(".7Z")
	assert.Equal(t, archive.Mime7z, s)

	s = archive.Extension(archive.Mime7z)
	assert.Equal(t, s, archive.Ext7z)

	s = archive.Extension(".tar.bz2")
	assert.Equal(t, archive.MimeTar, s)
}

func TestReadMIME(t *testing.T) {

	mime, err := archive.ReadMIME("")
	assert.NotNil(t, err)
	assert.Equal(t, "", mime)

	mime, err = archive.ReadMIME(mock.NoSuchFile)
	assert.NotNil(t, err)
	assert.Equal(t, "", mime)

	// test unsupported file types
	unsupported := []string{"txt", "xz"}
	for _, ext := range unsupported {
		s, err := mock.Extension(ext)
		assert.Nil(t, err)
		assert.NotEqual(t, "", s)
		_, err = archive.ReadMIME(s)
		assert.NotNil(t, err)
	}

	// test supported archives
	supported := []string{"7z", "zip"}
	for _, ext := range supported {
		s, err := mock.Extension(ext)
		assert.Nil(t, err)
		assert.NotEqual(t, "", s)
		mime, err = archive.ReadMIME(s)
		assert.Nil(t, err)
		assert.Contains(t, mime, "application/")
	}
}

func TestConfig_WalkArchiver(t *testing.T) {
	c := dupe.Config{Test: true}

	err := c.WalkArchiver(nil, "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = c.WalkArchiver(db, "")
	assert.NotNil(t, err)

	err = c.WalkArchiver(db, mock.NoSuchFile)
	assert.NotNil(t, err)

	item1, err := mock.Item(1)
	assert.Nil(t, err)
	err = c.WalkArchiver(db, parse.Bucket(item1))
	assert.Nil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)
	err = c.WalkArchiver(db, parse.Bucket(bucket1))
	assert.Nil(t, err)
}

func TestConfigRead7Zip(t *testing.T) {
	c := dupe.Config{Test: true, Quiet: false, Debug: false}

	err := c.Read7Zip(nil, "", "")
	assert.NotNil(t, err)

	db, err := mock.Database()
	assert.Nil(t, err)
	defer db.Close()

	err = c.Read7Zip(db, "", "")
	assert.NotNil(t, err)

	bucket1, err := mock.Bucket(1)
	assert.Nil(t, err)

	err = c.Read7Zip(db, parse.Bucket(bucket1), "")
	assert.NotNil(t, err)

	z7, err := mock.Extension("7z")
	assert.Nil(t, err)
	err = c.Read7Zip(db, parse.Bucket(z7), "")
	assert.NotNil(t, err)

	err = c.Read7Zip(db, parse.Bucket(bucket1), z7)
	assert.Nil(t, err)
}
