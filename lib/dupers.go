// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package dupers

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gookit/color"
)

type Config struct {
	Timer  time.Time
	Dirs   []string
	NoWalk bool
	Quiet  bool
	test   bool
	files  int // rename to files
	hashes hash
}

type (
	hash map[[32]byte]string
)

// Status summarizes the files scan.
func (c Config) Status() string {
	if c.Quiet {
		return ""
	}
	return "todo: status"
}

// WalkDirs walks the directories provided by the Arg slice for zip archives to extract any found comments.
func (c *Config) WalkDirs() {
	c.init()
	// walk through the directories provided
	for _, root := range c.Dirs {
		if err := c.WalkDir(root); err != nil {
			c.Error(err)
		}
	}
}

// WalkDir walks the root directory for zip archives and to extract any found comments.
func (c *Config) WalkDir(root string) error {
	c.init()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		// skip directories
		if d.IsDir() {
			return nil
		}
		// skip sub-directories
		if c.NoWalk && filepath.Dir(path) != filepath.Dir(root) {
			return nil
		}
		c.files++
		if !c.test && !c.Quiet {
			fmt.Print("\r", color.Secondary.Sprint("Scanned "), color.Primary.Sprintf("%d files", c.files))
		}

		// TODO: hash or read file

		// read zip file comment
		// cmmt, err := Read(path, c.Raw)
		// if err != nil {
		// 	c.Error(err)
		// 	return nil
		// }
		// if cmmt == "" {
		// 	return nil
		// }

		// hash the comment

		//hash := sha256.Sum256([]byte(strings.TrimSpace(cmmt)))
		hash, err := sum(path)
		if err != nil {
			return err
		}
		if hash == [32]byte{} {
			return nil
		}
		c.hashes[hash] = path

		return err
	})
	fmt.Printf("\n%+v\n", c.hashes)
	return err
}

// init initialise the Config maps.
func (c *Config) init() {
	if c.hashes == nil {
		c.hashes = make(hash)
	}
}

func sum(path string) (hash [32]byte, err error) {
	const Kb = 1024

	f, err := os.Open(path)
	if err != nil {
		return hash, err
	}
	defer f.Close()

	buf := make([]byte, Kb*Kb)

	h := sha256.New()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return hash, err
	}
	copy(hash[:], h.Sum(nil))

	return hash, nil
}
