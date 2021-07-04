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
	"strings"
	"sync"
	"time"

	"github.com/gookit/color"
)

type Config struct {
	Timer  time.Time
	Dirs   []string
	Scan   string
	NoWalk bool
	Quiet  bool
	test   bool
	files  int // rename to files
	hashes hash
	scans  hash
}

type (
	hash map[[32]byte]string
)

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

	var wg sync.WaitGroup // Move to Config via c.init()?
	//defer wg.Done()

	// var hasher = func(path string) {
	// 	defer wg.Done()
	// 	hash, err := sum(path)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	// 	if hash == [32]byte{} {
	// 		return
	// 	}
	// 	c.hashes[hash] = path
	// }

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
			case ".git", ".cache", ".config", ".local":
				return filepath.SkipDir
			default:
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !d.Type().IsRegular() {
			return nil
		}

		// skip hidden files and directories
		// if len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".") {
		// 	return nil
		// }
		// if len(path) > 1 && strings.Index(path, "/.") > 0 {
		// 	return nil
		// }

		// p := strings.Split(path, string(os.PathSeparator))
		// //fmt.Println(p)
		// for _, sub := range p {
		// 	if sub == "" {
		// 		continue
		// 	}
		// 	if sub[0:1] == "." {
		// 		return nil //filepath.SkipDir
		// 	}
		// }
		// skip sub-directories
		//c.NoWalk = true
		// fmt.Println(path)
		// fmt.Println(d.Type())
		// fmt.Println(filepath.Dir(path), "<-->", filepath.Dir(root))
		if c.NoWalk && filepath.Dir(path) != filepath.Dir(root) {
			return nil
		}
		c.files++
		if !c.test && !c.Quiet {
			fmt.Print("\u001b[2K")
			fmt.Print("\r", color.Secondary.Sprint("Scanned "), color.Primary.Sprintf("%d files:", c.files), " ", path)
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

		wg.Add(1)
		go c.hash(path, &wg)
		//go hasher(path)
		//hasher(path)
		wg.Wait()

		return err
	})
	//fmt.Printf("\n%+v\n", c.hashes)
	return err
}

// init initialise the Config maps.
func (c *Config) init() {
	if c.hashes == nil {
		c.hashes = make(hash)
	}
}

func (c *Config) hash(path string, wg *sync.WaitGroup) {
	defer wg.Done()
	hash, err := sum(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	if hash == [32]byte{} {
		return
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
