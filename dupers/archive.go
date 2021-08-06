// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
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
	"strings"
	"sync"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/out"
	"github.com/bodgit/sevenzip"
	"github.com/gookit/color"
	"github.com/h2non/filetype"
	"github.com/mholt/archiver"
	bolt "go.etcd.io/bbolt"
)

const (
	app7z  = "application/x-7z-compressed"
	appBZ2 = "application/x-bzip2"
	appGZ  = "application/gzip"
	appRAR = "application/vnd.rar"
	appTar = "application/x-tar"
	appXZ  = "application/x-xz"
	appZip = "application/zip"
	ext7z  = ".7z"
	oneKb  = 1024
	oneMb  = oneKb * oneKb
)

// extension finds either a compressed file extension or mime type and returns its match.
func extension(find string) string {
	// mime types refer to data types and do not contain encoding information
	// * mime not detected by h2non/filetype
	ext := map[string]string{
		ext7z:      app7z,
		".bz2":     appBZ2,
		".gz":      appGZ,
		".rar":     appRAR,
		".tar":     appTar,
		".zip":     appZip,
		".lz4":     "application/x-lz4",           // LZ4*
		".sz":      "application/x-snappy-framed", // Snappy*
		".xz":      appXZ,                         // XZ Utils
		".zst":     "application/zstd",            // Zstandard (zstd)*
		".tar.br":  appTar,                        // tar + Brotli compression
		".tbr":     appTar,                        //
		".tar.gz":  appTar,                        // tar + gzip compression
		".tgz":     appTar,                        //
		".tar.bz2": appTar,                        // tar + bzip2 compression
		".tbz2":    appTar,                        //
		".tar.xz":  appTar,                        // tar + XZ compression
		".txz":     appTar,                        //
		".tar.lz4": appTar,                        // tar + LZ4 compression
		".tlz4":    appTar,                        //
		".tar.sz":  appTar,                        // tar + snappy compression
		".tsz":     appTar,                        //
		".tar.zst": appTar,                        // tar + Zstandard compression
	}
	f := strings.ToLower(find)
	for k, v := range ext {
		if k == f {
			return v
		}
		if v == f {
			return k
		}
		if !strings.HasPrefix(find, ".") {
			if k == fmt.Sprintf(".%s", f) {
				return k
			}
		}
	}
	return ""
}

// IsArchive returns true if the read named file is compressed using a supported archive format.
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
		app7z,
		appBZ2,
		appGZ,
		appRAR,
		appTar,
		appZip:
		// supported archives
		return true, kind.MIME.Value, nil
	case
		appXZ, // not supported in archiver v3.5.0
		"application/vnd.ms-cab-compressed",
		"application/x-compress",
		"application/x-lzip",
		"application/x-unix-archive":
		// unsupported archives
		return false, kind.MIME.Value, nil
	}
	// non-archives
	return false, "", nil
}

// IsExtension returns true if the named file extension matches a supported archive format.
func IsExtension(name string) (result bool, mime string) {
	ext := filepath.Ext(name)
	if ext == "" {
		return false, ""
	}
	if find := extension(ext); find != "" {
		return true, find
	}
	return false, ""
}

// WalkArchiver walks the bucket directory saving the checksums of new files to the database.
// Any archived files supported by archiver will also have its content hashed.
// Archives within archives are currently left unwalked.
func (c *Config) WalkArchiver(name Bucket) error {
	if name == "" {
		return ErrNoBucket
	}
	root := string(name)
	c.init()
	skip := c.skipFiles()
	if c.db == nil {
		c.OpenDB()
		defer c.db.Close()
	}
	// create a new bucket if needed
	if err := c.createBucket(name); err != nil {
		return err
	}
	// get a list of all the bucket filenames
	if err := c.listItems(root); err != nil {
		out.ErrCont(err)
	}
	// walk the root directory of the bucket
	var wg sync.WaitGroup
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if c.Debug {
			out.Bug("walk file: " + path)
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if err1 := skipDir(d); err1 != nil {
			return err1
		}
		if skipFile(d.Name()) {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if skipSelf(path, skip...) {
			return nil
		}
		err = c.walkThread(root, path, &wg)
		if err != nil {
			out.ErrCont(err)
		}
		return nil
	})
	return err
}

func (c *Config) walkThread(bucket, path string, wg *sync.WaitGroup) error {
	// detect archive type by file extension
	ext := strings.ToLower(filepath.Ext(path))
	ok, _ := IsExtension(path)
	if c.Debug {
		out.Bug(fmt.Sprintf("is known extension: %v, %s", ok, ext))
	}
	if !ok {
		// detect archive type by mime type
		ok, mime, err := IsArchive(path)
		if err != nil {
			return err
		}
		if !ok {
			if c.Debug && mime != "" {
				s := fmt.Sprintf("archive not supported: %s: %s", mime, path)
				out.Bug(s)
			}
			return nil
		}
		ext = extension(mime)
	}
	c.files++
	if c.Debug {
		s := fmt.Sprintf("walkCompare #%d", c.files)
		out.Bug(s)
	}
	if errD := walkCompare(bucket, path, c); errD != nil {
		if !errors.Is(errD, ErrPathExist) {
			out.ErrFatal(errD)
		}
	}
	// multithread archive reader
	wg.Add(1)
	go func() {
		switch ext {
		case "":
			// not a supported archive, do nothing
		case ext7z:
			c.read7Zip(bucket, path)
		default:
			c.readArchiver(bucket, path, ext)
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
	}); errors.Is(err, bolt.ErrBucketNotFound) {
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
		out.ErrAppend(err)
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
			out.ErrAppend(err)
			continue
		}
		defer rc.Close()
		cnt++
		buf, h := make([]byte, oneMb), sha256.New()
		if _, err := io.CopyBuffer(h, rc, buf); err != nil {
			out.ErrAppend(err)
			continue
		}
		var sum checksum
		copy(sum[:], h.Sum(nil))
		if err := c.updateArchiver(fp, bucket, sum); err != nil {
			out.ErrAppend(err)
			continue
		}
	}
	if c.Debug && cnt > 0 {
		s := fmt.Sprintf("read %d items within the 7-Zip archive", cnt)
		out.Bug(s)
	}
}

// readArchiver opens the named file archive, hashes its content and saves those to the bucket.
func (c *Config) readArchiver(bucket, archive, ext string) { // nolint: gocyclo
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
	cnt, filename := 0, archive
	// get the format by filename extension
	if ext != "" {
		filename = ext
	}
	f, err := archiver.ByExtension(strings.ToLower(filename))
	if err != nil {
		out.ErrCont(err)
		return
	}
	// commented archives not supported in archiver v3.5.0
	switch f.(type) {
	case
		// *archiver.Brotli,
		*archiver.Bz2,
		*archiver.Gz,
		*archiver.Lz4,
		*archiver.Rar,
		*archiver.Snappy,
		*archiver.Tar,
		// *archiver.TarBrotli,
		*archiver.TarBz2,
		*archiver.TarGz,
		*archiver.TarLz4,
		*archiver.TarSz,
		*archiver.TarXz,
		// *archiver.TarZstd,
		*archiver.Xz,
		*archiver.Zip:
		w := f.(archiver.Walker)
		if err := w.Walk(archive, func(f archiver.File) error {
			if f.IsDir() {
				return nil
			}
			if !f.FileInfo.Mode().IsRegular() {
				return nil
			}
			if skipFile(f.Name()) {
				return nil
			}
			fp := filepath.Join(archive, f.Name())
			if c.findItem(fp) {
				return nil
			}
			buf, h := make([]byte, oneMb), sha256.New()
			if _, err := io.CopyBuffer(h, f, buf); err != nil {
				out.ErrAppend(err)
				return nil
			}
			var sum checksum
			copy(sum[:], h.Sum(nil))
			if err := c.updateArchiver(fp, bucket, sum); err != nil {
				out.ErrAppend(err)
			}
			cnt++
			return nil
		}); err != nil {
			out.ErrAppend(err)
		}
	default:
		color.Warn.Printf("Unsupported archive: '%s'\n", archive)
		return
	}
	if c.Debug && cnt > 0 {
		s := fmt.Sprintf("read %d items within the archive", cnt)
		out.Bug(s)
	}
}

// updateArchiver saves the checksum and path values to the bucket.
func (c *Config) updateArchiver(path, bucket string, sum checksum) error {
	if c.Debug {
		out.Bug("update archiver: " + path)
	}
	if c.db == nil {
		return bolt.ErrDatabaseNotOpen
	}
	if err := c.db.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(bucket))
		if b1 == nil {
			return bolt.ErrBucketNotFound
		}
		return b1.Put([]byte(path), sum[:])
	}); err != nil {
		return err
	}
	c.compare[sum] = path
	return nil
}
