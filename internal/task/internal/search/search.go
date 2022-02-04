package search

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/out"
)

var (
	ErrNoArgs = errors.New("request is missing arguments")
	ErrSearch = errors.New("search request needs an expression")
)

// CmdErr parses the arguments of the search command.
func CmdErr(l int, test bool) {
	if l > 1 {
		return
	}
	out.ErrCont(ErrSearch)
	fmt.Println("A search expression can be a partial or complete filename,")
	fmt.Println("or a partial or complete directory.")
	out.Example("\ndupers search <search expression> [optional, directories to search]")
	if !test {
		out.ErrFatal(nil)
	}
}

func Compare(f *cmd.Flags, term string, buckets []string) *database.Matches {
	var err error
	var m *database.Matches
	switch {
	case *f.Filename && !*f.Exact:
		if m, err = database.CompareBaseNoCase(term, buckets...); err != nil {
			Error(err, false)
		}
	case *f.Filename && *f.Exact:
		if m, err = database.CompareBase(term, buckets...); err != nil {
			Error(err, false)
		}
	case !*f.Filename && !*f.Exact:
		if m, err = database.CompareNoCase(term, buckets...); err != nil {
			Error(err, false)
		}
	case !*f.Filename && *f.Exact:
		if m, err = database.Compare(term, buckets...); err != nil {
			Error(err, false)
		}
	}
	return m
}

// Error parses the errors from search compares.
func Error(err error, test bool) {
	if errors.Is(err, database.ErrDBEmpty) {
		out.ErrCont(err)
		return
	}
	if errors.As(err, &database.ErrBucketNotFound) {
		out.ErrCont(err)
		fmt.Println("\nTo add this directory to the database, run:")
		dir := err.Error()
		if errors.Unwrap(err) == nil {
			s := fmt.Sprintf("%s: ", errors.Unwrap(err))
			dir = strings.ReplaceAll(err.Error(), s, "")
		}
		s := fmt.Sprintf("dupers up %s\n", dir)
		out.Example(s)
		if !test {
			out.ErrFatal(nil)
		}
	}
	if !test {
		out.ErrFatal(err)
	}
}
