// © Ben Garrett https://github.com/bengarrett/dupers

// dupers todo
package dupers

import (
	"fmt"
	"net/http"
	"strings"

	str "github.com/boyter/go-string"
	"github.com/gookit/color"
	"golang.org/x/tools/godoc/util"
)

//type hash map[[32]byte]string

// type (
// 	TextMatch map[string]int
// )

type TextMatch struct {
	Text string
	Line int
}

const (
	textBucket = "_?"
)

func Highlight(needle string, buf *[]byte) string {
	all := str.IndexAllIgnoreCase(string(*buf), needle, -1)
	if len(all) > 0 {
		return str.HighlightString(string(*buf), all, ">", "<")
	}
	return ""
}

func Highlights(needle string, mark bool, buf *[]byte) []TextMatch {
	//matches, tm := []TextMatch{}, TextMatch{}
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

func Search(needle string) {}
