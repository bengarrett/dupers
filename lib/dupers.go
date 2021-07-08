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

	"github.com/bengarrett/dupers/lib/database"
	"github.com/dustin/go-humanize"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
)

type Config struct {
	Timer   time.Time
	Buckets []string
	Bucket  string
	Lookup  bool // todo
	NoWalk  bool // todo
	Quiet   bool // todo
	test    bool // todo
	files   int
	db      *bolt.DB
	compare hash // database or file system hashes
	queries hash // user scan or lookup requests hashes
}

type (
	hash map[[32]byte]string
)

var ErrPathExist = errors.New("path exists in the database bucket")

// init initialises the Config maps and database.
func (c *Config) init() {
	// normalise bucket names
	for i, b := range c.Buckets {
		abs, err := filepath.Abs(b)
		if err != nil {
			log.Println(err)
			c.Buckets[i] = ""
			continue
		}
		c.Buckets[i] = abs
	}
	if c.compare == nil {
		c.compare = make(hash)
	}
}

func (c *Config) Query() {
	c.init()
	for hash, path := range c.queries {
		s := "\n"
		s += color.Info.Sprint("Looking up") +
			":" +
			fmt.Sprintf("\t%s", path)
		fmt.Print(s)
		var (
			err   error
			finds []string
		)
		for _, bucket := range c.Buckets {
			finds, c.files, err = database.Seek(hash, bucket)
			if err != nil {
				log.Println(err)
				continue
			}
		}
		if len(finds) > 0 {
			verb := "duplicate matches"
			if len(finds) == 1 {
				verb = "a duplicate match"
			}
			s := "\r"
			s += color.Info.Sprintf("Found %s", verb) +
				":" +
				fmt.Sprintf(" %s", path)
			fmt.Print(s)
			for _, find := range finds {
				fmt.Println(matchItem(find))
			}
		}
	}

	//c.queries[s] = path

	// TODO: SEARCH
	// if err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
	// 	// skip directories
	// 	if d.IsDir() {
	// 		switch d.Name() {
	// 		// the SkipDir return tells WalkDir to skip all files in these directories
	// 		case ".git", ".cache", ".config", ".local", "node_modules":
	// 			return filepath.SkipDir
	// 		default:
	// 			if strings.HasPrefix(d.Name(), ".") {
	// 				return filepath.SkipDir
	// 			}
	// 			return nil
	// 		}
	// 	}
	// 	// skip non-files such as symlinks
	// 	if !d.Type().IsRegular() {
	// 		return nil
	// 	}
	// 	// walk through the directories provided
	// 	fmt.Println("Looking up:", path)
	// 	for _, bucket := range c.Buckets {
	// 		if err := database.Seek(path, bucket); err != nil {
	// 			log.Println(err)
	// 		}
	// 	}
	// 	return nil
	// }); err != nil {
	// 	log.Fatalln(err)
	// }

	// for h, path := range c.queries {

	// }
}

func (c *Config) Matches() {
	for h, path := range c.queries {
		l := c.lookupOne(h)
		if l == "" {
			continue
		}
		if l == path {
			continue
		}
		fmt.Println(match(path, l))
	}
}

func (c *Config) lookupOne(h [32]byte) string {
	if f := c.compare[h]; f != "" {
		return f
	}
	return ""
}

func match(path, match string) string {
	s := "\n"
	s += color.Info.Sprint("Found duplicate match") +
		":" +
		fmt.Sprintf("\t%s", path) +
		matchItem(match)
	return s
}

func matchItem(match string) string {
	s := color.Success.Sprint("\n  ⤷ ") +
		fmt.Sprint(match)
	stat, err := os.Stat(match)
	if err != nil {
		return s
	}
	s += "\n    " +
		fmt.Sprintf("%s, ", stat.ModTime().Format("02 Jan 2006 15:04")) +
		humanize.Bytes(uint64(stat.Size()))
	return s
}

func (c *Config) Queries() {
	if c.queries == nil {
		c.queries = make(hash)
	}

	root, err := filepath.Abs(c.Bucket)
	if err != nil {
		log.Fatalln(err)
	}

	stat, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalln("path does not exist:", root)
	} else if err != nil {
		log.Fatalln(err)
	}

	if !stat.IsDir() {
		if err = c.store(root); err != nil {
			log.Fatalln(err)
		}
		return
	}

	if err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// skip directories
		if d.IsDir() {
			switch d.Name() {
			// the SkipDir return tells WalkDir to skip all files in these directories
			case ".git", ".cache", ".config", ".local", "node_modules":
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
		return c.store(path)
	}); err != nil {
		log.Fatalln(err)
	}
}

func (c *Config) store(path string) error {
	s, err := sum(path)
	if err != nil {
		return err
	}
	c.queries[s] = path
	return nil
}

// Search
func (c Config) Search(s string) {
	// if err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
	// 	// skip directories
	// 	if d.IsDir() {
	// 		switch d.Name() {
	// 		// the SkipDir return tells WalkDir to skip all files in these directories
	// 		case ".git", ".cache", ".config", ".local", "node_modules":
	// 			return filepath.SkipDir
	// 		default:
	// 			if strings.HasPrefix(d.Name(), ".") {
	// 				return filepath.SkipDir
	// 			}
	// 			return nil
	// 		}
	// 	}
	// 	// skip non-files such as symlinks
	// 	if !d.Type().IsRegular() {
	// 		return nil
	// 	}
	// 	// walk through the directories provided
	// 	fmt.Println("Looking up:", path)
	// 	for _, bucket := range c.Buckets {
	// 		if err := database.Seek(path, bucket); err != nil {
	// 			log.Println(err)
	// 		}
	// 	}
	// 	return nil
	// }); err != nil {
	// 	log.Fatalln(err)
	// }
}

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

// WalkDirs walks the directories provided by the Arg slice for zip archives to extract any found comments.
func (c *Config) WalkDirs() {
	c.init()
	// walk through the directories provided
	for _, bucket := range c.Buckets {
		if err := c.WalkDir(bucket); err != nil {
			c.Error(err)
		}
	}
}

// WalkDir walks the root directory for zip archives and to extract any found comments.
func (c *Config) WalkDir(root string) error {
	c.init()
	skip := c.skipFiles()
	fmt.Println(skip)

	if c.db == nil {
		name, err := database.DB()
		if err != nil {
			log.Fatalln(err)
		}
		if c.db, err = bolt.Open(name, database.FileMode, nil); err != nil {
			log.Fatalln(err)
		}
		defer c.db.Close()
	}

	if err := c.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(root)); b == nil {
			fmt.Println("Creating a database bucket for", root)
			_, err1 := tx.CreateBucket([]byte(root))
			return err1
		}
		return nil
	}); err != nil {
		log.Fatalln(err)
	}

	var wg sync.WaitGroup
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
		// user flag, skip recursive directories
		if c.NoWalk && filepath.Dir(path) != filepath.Dir(root) {
			return nil
		}
		// skip self file matches
		//fmt.Println("\n->", path)
		for _, n := range skip {
			if path == n {
				return nil
			}
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
				c.compare[hash] = path
				return ErrPathExist
			}
			return nil
		})
		if errDB != nil {
			if errDB == ErrPathExist {
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
	c.compare[hash] = path

}

func (c *Config) skipFiles() (files []string) {
	for _, path := range c.queries {
		files = append(files, path)
	}
	return files
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
