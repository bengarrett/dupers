// © Ben Garrett https://github.com/bengarrett/dupers
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
	ErrNoArgs  = errors.New("request is missing arguments")
	ErrNoFlags = errors.New("no command flags provided")
	ErrSearch  = errors.New("search request needs an expression")
)

// CmdErr parses the arguments of the search command.
func CmdErr(l int, test bool) error {
	if l > 1 {
		return nil
	}
	out.ErrCont(ErrSearch)
	fmt.Println("A search expression can be a partial or complete filename,")
	fmt.Println("or a partial or complete directory.")
	out.Example("\ndupers search <search expression> [optional, directories to search]")
	if test {
		return ErrSearch
	}
	out.ErrFatal(nil)
	return nil
}

func Compare(f *cmd.Flags, term string, buckets []string, test bool) (*database.Matches, error) {
	var err error
	var m *database.Matches
	if f == nil {
		return nil, ErrNoFlags
	}
	switch {
	case *f.Filename && !*f.Exact:
		if m, err = database.CompareBaseNoCase(term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	case *f.Filename && *f.Exact:
		if m, err = database.CompareBase(term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	case !*f.Filename && !*f.Exact:
		if m, err = database.CompareNoCase(term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	case !*f.Filename && *f.Exact:
		if m, err = database.Compare(term, buckets...); err != nil {
			return nil, Error(err, test)
		}
	}
	return m, nil
}

// Error parses the errors from search compares.
func Error(err error, test bool) error {
	if errors.Is(err, database.ErrDBEmpty) {
		out.ErrCont(err)
		return nil
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
		if test {
			return nil
		}
		out.ErrFatal(nil)
	}
	if test {
		return err
	}
	out.ErrFatal(err)
	return nil
}
