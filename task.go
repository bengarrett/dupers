// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/dupers"
	"github.com/bengarrett/dupers/out"
	"github.com/gookit/color"
)

func checkBkt(term, cmd, name string) {
	if name != "" {
		return
	}
	out.ErrCont(ErrDatabaseName)
	fmt.Printf("Cannot %s the bucket as no bucket name was provided.\n", term)
	out.Example(fmt.Sprintf("\ndupers %s <bucket name>", cmd))
	out.ErrFatal(nil)
}

func exportBucket(quiet bool, args [2]string) {
	checkBkt(dex, dex, args[1])
	name, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	if errEx := database.Exist(name, nil); errors.Is(errEx, database.ErrBucketNotFound) {
		out.ErrCont(errEx)
		fmt.Printf("Bucket name: %s\n", name)
		out.Example("\ndupers export <bucket name>")
		out.ErrFatal(nil)
	} else if errEx != nil {
		out.ErrFatal(errEx)
	}
	exp, errEx := database.ExportCSV(name, nil)
	if errEx != nil {
		out.ErrFatal(errEx)
	}
	s := fmt.Sprintf("The exported bucket file is at: %s", exp)
	out.Response(s, quiet)
}

func listBucket(quiet bool, args [2]string) {
	checkBkt("list", dls, args[1])
	name, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	ls, err := database.List(name, nil)
	if err != nil {
		out.ErrCont(err)
	}
	// sort the filenames
	var names []string
	for name := range ls {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		sum := ls[database.Filepath(name)]
		fmt.Printf("%x %s\n", sum, name)
	}
	if cnt := len(ls); !quiet && cnt > 0 {
		fmt.Printf("%s %s\n", color.Primary.Sprint(cnt),
			color.Secondary.Sprint("items listed. Checksums are 32 byte, SHA-256 (FIPS 180-4)."))
	}
}

func moveBucket(quiet bool, args [3]string) {
	b, dir := args[1], args[2]
	checkBkt("move and rename", dmv, b)
	name, err := filepath.Abs(b)
	if err != nil {
		out.ErrFatal(err)
	}
	if errEx := database.Exist(name, nil); errors.Is(errEx, database.ErrBucketNotFound) {
		out.ErrCont(errEx)
		fmt.Printf("Bucket name: %s\n", name)
		out.Example("\ndupers mv <bucket name> <new directory>")
		out.ErrFatal(nil)
	} else if errEx != nil {
		out.ErrFatal(errEx)
	}
	if dir == "" {
		fmt.Println("Cannot move and rename bucket in the database as no new directory was provided.")
		out.Example(fmt.Sprintf("\ndupers mv %s <new directory>", b))
		out.ErrFatal(nil)
	}
	newName, err := filepath.Abs(dir)
	if err != nil {
		out.ErrFatal(err)
	}
	if newName == "" {
		out.ErrFatal(ErrNewName)
	}
	if !quiet {
		fmt.Printf("Current:\t%s\nNew path:\t%s\n", name, newName)
		fmt.Println("This only renames the bucket, it does not move files on your system.")
		if !out.YN("Rename bucket") {
			return
		}
	}
	if err := database.Rename(name, newName); err != nil {
		out.ErrFatal(err)
	}
}

func removeBucket(quiet bool, args [2]string) {
	checkBkt("remove", drm, args[1])
	name, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	if err := database.RM(name); err != nil {
		if errors.Is(err, database.ErrBucketNotFound) {
			// retry with the original argument
			if err1 := database.RM(args[1]); err1 != nil {
				if errors.Is(err1, database.ErrBucketNotFound) {
					out.ErrCont(err1)
					fmt.Printf("Bucket to remove: %s\n", color.Danger.Sprint(name))
					buckets, err2 := database.AllBuckets(nil)
					if err2 != nil {
						out.ErrFatal(err2)
					}
					fmt.Printf("Buckets in use:   %s\n", strings.Join(buckets, "\n\t\t  "))
					out.ErrFatal(nil)
				}
				out.ErrFatal(err1)
			}
		}
	}
	s := fmt.Sprintf("Removed bucket from the database: '%s'\n", name)
	out.Response(s, quiet)
}

func rescanBucket(c *dupers.Config, plus bool, args [2]string) {
	cmd := dup
	if plus {
		cmd = dupp
	}
	checkBkt("add or update", cmd, args[1])
	path, err := filepath.Abs(args[1])
	if err != nil {
		out.ErrFatal(err)
	}
	name := dupers.Bucket(path)
	if plus {
		if err := c.WalkArchiver(name); err != nil {
			out.ErrFatal(err)
		}
	} else if err := c.WalkDir(name); err != nil {
		out.ErrFatal(err)
	}
	if !c.Quiet {
		if c.Timer() > winRemind {
			fmt.Printf("\n%s: %s\n", perfMsg, color.Debug.Sprintf("duper -quiet %s ...", cmd))
		}
		fmt.Println(c.Status())
	}
}
