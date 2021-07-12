// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package dupers

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/bengarrett/dupers/lib/database"
	str "github.com/boyter/go-string"
	"github.com/gookit/color"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/tools/godoc/util"
)

type TextMatch struct {
	Text string
	Line int
}

const textBucket = "_?text"

func Finds(needle, path string, buf []byte) int {
	finds := Highlights(needle, true, &buf)
	l := len(finds)
	if l > 0 {
		o := "occurrences"
		if l == 1 {
			o = "occurrence"
		}
		s := color.Success.Sprintf("Found %d %s of '%s' in: ", l, o, needle)
		fmt.Printf("\n%s%s\n\n", s, path)
		for i, f := range finds {
			s = color.Secondary.Sprintf("%d. Line %4d: ", i+1, f.Line)
			fmt.Printf("%s%s\n", s, f.Text)
		}
	}
	return l
}

func Highlight(needle string, buf *[]byte) string {
	all := str.IndexAllIgnoreCase(string(*buf), needle, -1)
	if len(all) > 0 {
		return Mark(string(*buf), all)
	}
	return ""
}

func Highlights(needle string, mark bool, buf *[]byte) []TextMatch {
	tm := []TextMatch{}
	lines := strings.Split(string(*buf), "\n")
	cnt := 0
	for i, line := range lines {
		all := str.IndexAllIgnoreCase(string(line), needle, -1)
		if len(all) == 0 {
			continue
		}
		cnt++
		if mark {
			line = Mark(line, all)
		}
		t := TextMatch{line, i}
		tm = append(tm, t)
	}
	return tm
}

func Mark(s string, locations [][]int) string {
	m := ""
	for _, l := range locations {
		a, b := l[0], l[1]
		pre, mid, suf := s[:a], s[a:b], s[b:]
		m += fmt.Sprintf("%s%s%s", pre, color.Yellow.Sprint(mid), suf)
	}
	return m
}

// todo: check each 8-bit character values to be in the range of 0-255.
func Is8BitText(buf *[]byte) bool {
	const oneKb = 1000
	for i, b := range *buf {
		if i > oneKb {
			return true
		}
		// unicode.MaxASCII || !unicode.IsPrint(rune(b))
		if b > 255 {
			return false
		}
	}
	return true
}

func IsText(buf *[]byte) bool {
	if util.IsText(*buf) {
		return true
	}
	if strings.HasPrefix(http.DetectContentType(*buf), "text/plain;") {
		return true
	}
	if Is8BitText(buf) {
		return true
	}
	return false
}

func (c *Config) Search(needle string) {
	name, err := database.DB()
	if err != nil {
		log.Fatalln(err)
	}
	db, err := bolt.Open(name, database.FileMode, &bolt.Options{ReadOnly: true})
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	cnt := 0
	if err1 := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(textBucket))
		if b == nil {
			return database.ErrNoBucket
		}
		err = b.ForEach(func(key, b []byte) error {
			if f := Finds(needle, string(key), b); f > 0 {
				cnt += f
			}
			return nil
		})
		return err
	}); err1 != nil {
		log.Fatalln(err1)
	}
	if cnt == 0 {
		color.Secondary.Printf("no results for '%s'\n", needle)
		return
	}
	s := "\n"
	s += color.Secondary.Sprint("Found ") +
		color.Primary.Sprintf("%d matches", cnt)
	s += color.Secondary.Sprint(", taking ") +
		color.Primary.Sprintf("%s", time.Since(c.Timer))
	if runtime.GOOS != winOS {
		s += "\n"
	}
	fmt.Println(s)
}
