// © Ben Garrett https://github.com/bengarrett/dupers

// Dupers is the blazing-fast file duplicate checker and filename search.
package dupers

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/bengarrett/dupers/lib/database"
	"github.com/bengarrett/dupers/lib/out"
	"github.com/bodgit/sevenzip"
	"github.com/gookit/color"
	"github.com/h2non/filetype"
	"github.com/mholt/archiver"
	bolt "go.etcd.io/bbolt"
)

const x7z = "application/x-7z-compressed"

// IsArchive returns true if the named file is compressed using a supported archive format.
func IsArchive(name string) (result bool, mime string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return false, "", err
	}
	defer f.Close()
	kind, err := filetype.MatchReader(f)
	if err != nil {
		return false, "", err
	}
	switch kind.MIME.Value {
	case
		"application/gzip",
		"application/vnd.rar",
		x7z,
		"application/x-tar",
		"application/zip":
		return true, kind.MIME.Value, nil
	case
		"application/x-bzip2",
		"application/x-xz",
		"application/vnd.ms-cab-compressed",
		"application/x-unix-archive",
		"application/x-compress",
		"application/x-lzip":
		return false, kind.MIME.Value, nil
	}
	return false, "", nil
}

// WalkArchiver walks the bucket directory saving the hash values of new files to the database.
// Any archived files supported by archiver will also have its content hashed.
// Archives within archives are currently left unwalked.
func (c *Config) WalkArchiver(bucket string) error {
	c.init()
	skip := c.skipFiles()
	// open database
	if c.db == nil {
		name, err := database.DB()
		if err != nil {
			out.ErrFatal(err)
		}
		if c.db, err = bolt.Open(name, database.FileMode, nil); err != nil {
			out.ErrFatal(err)
		}
		defer c.db.Close()
	}
	// get a list of all the bucket filenames
	if err := c.listItems(bucket); err != nil {
		out.ErrCont(err)
	}
	// walk the root directory of the bucket
	var wg sync.WaitGroup
	err := filepath.WalkDir(bucket, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			out.Bug("walk file: " + path)
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		// skip directories
		if err1 := skipDir(d, true); err1 != nil {
			return err1
		}
		// skip files
		if skipFile(d) {
			return nil
		}
		// skip non-files such as symlinks
		if !d.Type().IsRegular() {
			return nil
		}
		// skip self file matches
		if skipSelf(path, skip) {
			return nil
		}
		// walk the directories
		return c.walkThread(bucket, path, &wg)
	})
	return err
}

func (c *Config) walkThread(bucket, path string, wg *sync.WaitGroup) error {
	// detect filetype
	ok, aType, err := IsArchive(path)
	if err != nil {
		return err
	}
	if !ok {
		if c.Debug && aType != "" {
			s := fmt.Sprintf("archive not supported: %s: %s", aType, path)
			out.Bug(s)
		}
		return nil
	}
	c.files++
	if c.Debug {
		s := fmt.Sprintf("walkDir #%d", c.files)
		out.Bug(s)
	}
	if errD := walkDir(bucket, path, c); errD != nil {
		if !errors.Is(errD, ErrPathExist) {
			out.ErrFatal(errD)
		}
	}
	// multithread archive reader
	wg.Add(1)
	go func() {
		switch aType {
		case x7z:
			c.read7Zip(bucket, path)
		default:
			c.readArchiver(bucket, path)
		}
		wg.Done()
	}()
	wg.Wait()
	return nil
}

// findItem returns true if the absolute file path is in c.sources.
func (c *Config) findItem(abs string) bool {
	for _, item := range c.sources {
		if item == abs {
			return true
		}
	}
	return false
}

// listItems sets c.sources to list all the filenames used in the bucket.
func (c *Config) listItems(bucket string) error {
	if c.Debug {
		out.Bug("list bucket items: " + bucket)
	}
	abs, err := database.Abs(bucket)
	if err != nil {
		out.ErrCont(err)
	}
	if err = c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(abs)
		if b == nil {
			return nil
		}
		err = b.ForEach(func(key, _ []byte) error {
			if bytes.Contains(key, []byte(bucket)) {
				c.sources = append(c.sources, string(database.Filepath(key)))
			}
			return nil
		})
		return err
	}); errors.Is(err, database.ErrNoBucket) {
		return fmt.Errorf("%w: '%s'", err, abs)
	} else if err != nil {
		return err
	}
	return nil
}

// read7Zip opens the named 7-Zip archive, hashes its content and saves those to the bucket.
func (c *Config) read7Zip(bucket, name string) {
	if c.Debug {
		out.Bug("read 7zip: " + name)
	}
	r, err := sevenzip.OpenReader(name)
	if err != nil {
		out.ErrCont(err)
		return
	}
	defer r.Close()
	cnt := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		fp := filepath.Join(name, f.Name)
		if c.findItem(fp) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			out.ErrCont(err)
			continue
		}
		defer rc.Close()
		cnt++
		const oneKb = 1024
		buf, h := make([]byte, oneKb*oneKb), sha256.New()
		if _, err := io.CopyBuffer(h, rc, buf); err != nil {
			out.ErrCont(err)
			continue
		}
		hash := [32]byte{}
		copy(hash[:], h.Sum(nil))
		if err := c.updateArchiver(fp, bucket, hash); err != nil {
			out.ErrCont(err)
			continue
		}
	}
	if c.Debug && cnt > 0 {
		s := fmt.Sprintf("read %d items within the 7-Zip archive", cnt)
		out.Bug(s)
	}
}

// readArchiver opens the named file archive, hashes its content and saves those to the bucket.
func (c *Config) readArchiver(bucket, archive string) {
	const oneKb = 1024
	if c.Debug {
		out.Bug("read archiver: " + archive)
	}
	// catch any archiver panics such as as opening unsupported ZIP compression formats
	defer func() {
		if err := recover(); err != nil {
			if !c.Quiet {
				if !c.Debug {
					fmt.Println()
				}
				color.Warn.Printf("Unsupported archive: '%s'\n", archive)
			}
			if c.Debug {
				out.Bug(fmt.Sprint(err))
			}
		}
	}()
	cnt := 0
	if err := archiver.Walk(archive, func(f archiver.File) error {
		if f.IsDir() {
			return nil
		}
		fp := filepath.Join(archive, f.Name())
		if c.findItem(fp) {
			return nil
		}
		cnt++

		buf, h := make([]byte, oneKb*oneKb), sha256.New()
		if _, err := io.CopyBuffer(h, f, buf); err != nil {
			out.ErrCont(err)
			return nil
		}
		hash := [32]byte{}
		copy(hash[:], h.Sum(nil))
		if err := c.updateArchiver(fp, bucket, hash); err != nil {
			out.ErrCont(err)
		}
		return nil
	}); err != nil {
		out.ErrCont(err)
	}
	if c.Debug && cnt > 0 {
		s := fmt.Sprintf("read %d items within the archive", cnt)
		out.Bug(s)
	}
}

// updateArchiver saves the hash and path values to the bucket.
func (c *Config) updateArchiver(path, bucket string, hash [32]byte) error {
	if c.Debug {
		out.Bug("update archiver: " + path)
	}
	if err := c.db.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(bucket))
		return b1.Put([]byte(path), hash[:])
	}); err != nil {
		return err
	}
	c.compare[hash] = path
	return nil
}
