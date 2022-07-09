// © Ben Garrett https://github.com/bengarrett/dupers

package dupers_test

import (
	"fmt"
	"log"

	"github.com/bengarrett/dupers"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/task"
)

func Example_dupe() {
	dupe := dupers.Config{
		Debug: false, // debug output
		Quiet: false, // less verbose output
	}

	// directory or a file to check
	dupe.SetToCheck("test/files_to_check")

	// directories containing files to match against.
	// a bucket represents a directory path, it is used as the
	// database table name and contains the file metadata
	buckets := []string{"test/bucket1", "test/bucket2"}
	dupe.SetBucket(buckets...)

	if checkDirs, fc, bc := dupe.CheckPaths(); checkDirs {
		fmt.Printf("will lookup %d files in %d buckets\n", fc, bc)
	}

	// files or directories to compare (these are not saved to database)
	if err := dupe.WalkSource(); err != nil {
		log.Fatal(err)
	}

	// walk the bucket directories for new or changed files
	const fastMethod = false
	task.Lookup(&dupe, fastMethod)

	// print the found dupes and summaries
	fmt.Println(dupe.Print(), dupe.Status())
}

func Example_search() {
	const (
		term = ".zip" // partial or full name of a file or directory
		fn   = true   // match only filenames
		ex   = false  // exact case matching
	)

	// directories containing files to match against.
	// a bucket represents a directory path, it is used as the
	// database table name and contains the file metadata
	buckets := []string{"test/bucket1", "test/bucket2"}

	// search and compare the term against those in the buckets
	s := task.Search{
		Term:     term,
		Filename: fn,
		Exact:    ex,
		Buckets:  buckets,
	}
	m, err := s.Compare()
	if err != nil {
		log.Fatal(err)
	}

	// print the matches
	r := dupers.Print(false, ex, term, m)
	fmt.Println(r)

	// print a summary of the results
	total := len(*m)
	sum := cmd.SearchSummary(total, term, ex, fn)
	fmt.Println(sum)
}
