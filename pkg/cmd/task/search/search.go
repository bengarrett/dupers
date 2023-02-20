// © Ben Garrett https://github.com/bengarrett/dupers
package search

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/bengarrett/dupers/pkg/cmd"
	"github.com/bengarrett/dupers/pkg/database"
	bolt "go.etcd.io/bbolt"
)

var (
	ErrNoArgs  = errors.New("request is missing arguments")
	ErrNoFlags = errors.New("no command flags provided")
	ErrSearch  = errors.New("search request needs an expression")
)

// CmdErr parses the arguments of the search command.
func CmdErr(lenArgs int, test bool) error {
	if lenArgs > 1 {
		return nil
	}
	w := os.Stderr
	out.StderrCR(ErrSearch)
	fmt.Fprintln(w, "A search expression can be a partial or complete filename,")
	fmt.Fprintln(w, "or a partial or complete directory.")
	out.Example("\ndupers search <search expression> [optional, directories to search]")
	if !test {
		out.ErrFatal(nil)
	}
	return ErrSearch
}

func Compare(db *bolt.DB, f *cmd.Flags, term string, buckets []string, test bool) (*database.Matches, error) {
	if db == nil {
		return nil, bolt.ErrDatabaseNotOpen
	}
	if f == nil {
		return nil, ErrNoFlags
	}
	if f.Filename == nil || f.Exact == nil {
		return nil, ErrNoFlags
	}
	var err error
	var m *database.Matches
	switch {
	case *f.Filename && !*f.Exact:
		if m, err = database.CompareBaseNoCase(db, term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	case *f.Filename && *f.Exact:
		if m, err = database.CompareBase(db, term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	case !*f.Filename && !*f.Exact:
		if m, err = database.CompareNoCase(db, term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	case !*f.Filename && *f.Exact:
		if m, err = database.Compare(db, term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	}
	return m, nil
}

// Error parses the errors from search compares.
func Error(err error, test bool) error {
	if errors.Is(err, database.ErrEmpty) {
		out.StderrCR(err)
		return nil
	}
	if errors.Is(err, bolt.ErrBucketNotFound) {
		out.StderrCR(err)
		fmt.Fprintln(os.Stdout, "\nTo add this directory to the database, run:")
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
		return nil
	}
	if !test {
		out.ErrFatal(err)
	}
	return err
}
