// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package dupers

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

type Config struct {
	Timer  time.Time
	Dirs   []string
	Scan   string
	NoWalk bool
	Quiet  bool
	test   bool
	files  int
	hashes hash
	scans  hash
	db     *bolt.DB
}

type (
	hash map[[32]byte]string
)

var ExistingPath = errors.New("blah blah blah")

// Status summarizes the files scan.
func (c Config) Status() string {
	if c.Quiet {
		return ""
	}
	s := "\n"
	s += color.Secondary.Sprint("Scanned ") +
		color.Primary.Sprintf("%d files", c.files)
	if !c.test {
		s += color.Secondary.Sprint(", taking ") +
			color.Primary.Sprintf("%s", time.Since(c.Timer)) + "\n"
	}
	return s
}

func (c *Config) ScanDirs() {
	s, err := sum(c.Scan)
	if err != nil {
		fmt.Println("error: ", err)
	}
	fmt.Println("scan results for:", c.Scan)
	if f := c.hashes[s]; f != "" {
		fmt.Println("found duplicate: ", f)
	} else {
		fmt.Println("no duplicates found")
	}
	// for i, _ := range c.hashes {
	// 	if c.hashes[i] != "" {
	// 		fmt.Println(c.hashes[])
	// 		//fmt.Println("found duplicate:", c.Scan, "matches", n, ">>", c.hashes[i])
	// 	}
	// 	//fmt.Println(i, ":", n)
	// }
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

	if c.db == nil {
		var err error
		c.db, err = bolt.Open("my.db", 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer c.db.Close()
	}

	var wg sync.WaitGroup // Move to Config via c.init()?

	errDB := c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(root))
		if b == nil {
			fmt.Println("Creating a database bucket for", root)
			_, err := tx.CreateBucket([]byte(root))
			return err
		}
		return nil
	})
	if errDB != nil {
		log.Fatalln(errDB)
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		// skip directories
		if d.IsDir() {
			switch d.Name() {
			// the SkipDir return tells WalkDir to skip all files in these directories
			case ".git", ".cache", ".config", ".local":
				return filepath.SkipDir
			default:
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
		}
		// skip non-files such as symlinks
		if !d.Type().IsRegular() {
			return nil
		}

		if c.NoWalk && filepath.Dir(path) != filepath.Dir(root) {
			return nil
		}
		c.files++

		errDB := c.db.View(func(tx *bolt.Tx) error {
			if !c.test && !c.Quiet {
				fmt.Print("\u001b[2K")
				fmt.Print("\r", color.Secondary.Sprint("Looked up "), color.Primary.Sprintf("%d files:", c.files), " ", path)
			}
			b := tx.Bucket([]byte(root))
			if b == nil {
				return nil
			}
			h := b.Get([]byte(path))
			if len(h) > 0 {
				var hash [32]byte
				copy(hash[:], h)
				c.hashes[hash] = path
				return ExistingPath
			}
			return nil
		})
		if errDB != nil {
			if errDB == ExistingPath {
				return nil
			}
			log.Fatalln(errDB)
		}

		if !c.test && !c.Quiet {
			fmt.Print("\u001b[2K")
			fmt.Print("\r", color.Secondary.Sprint("Scanned "), color.Primary.Sprintf("%d files:", c.files), " ", path)
		}
		// hash the file
		wg.Add(1)
		go c.hash(path, root, &wg)
		wg.Wait()

		return err
	})
	return err
}

// init initialises the Config maps and database.
func (c *Config) init() {
	if c.hashes == nil {
		c.hashes = make(hash)
	}
}

func (c *Config) hash(path, root string, wg *sync.WaitGroup) {
	defer wg.Done()
	hash, err := sum(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	if hash == [32]byte{} {
		return
	}
	if err = c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(root))
		return b.Put([]byte(path), hash[:])
	}); err != nil {
		log.Println(err)
	}
	c.hashes[hash] = path

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
