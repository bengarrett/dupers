package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	Arg  []string
	Args []Arg
)

const (
	repoRoot = "./.."
)

func main() {
	path, err := filepath.Abs(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	args := runargs()
	rungo(path, args)
}

func rungo(path string, args Args) {
	ttl := len(args)
	for i, arg := range args {
		cnt := i + 1
		fmt.Fprintf(os.Stdout, "[%d of %d] running the following:\ngo %s\n\n",
			cnt, ttl, strings.Join(arg, " "))
		cmd := exec.Command("go", arg...)
		cmd.Dir = path
		p, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Fprintln(os.Stdout, string(p))
	}
}

func runmainwithrace() Arg {
	return Arg{"run", "-race", "main.go"}
}

func runargs() Args {
	runr := runmainwithrace()
	return Args{
		runr,
		append(runr, "-mono"),
		append(runr, "-h", "database"),
		append(runr, "search", "--help"),
		append(runr, "-help", "dupe"),
		append(runr, "-version"),
		append(runr, "-mono", "-version"),
		append(runr, "-mono", "-version", "-quiet"),
		append(runr, "-m", "-v", "-q"),
		append(runr, "database"),
		append(runr, "db", "-d", "-m"),
		append(runr, "-m", "-q", "-d", "-v", "-h", "-e", "-n", "-f", "-y"),
		append(runr, "-m", "-q", "-d", "-v", "-h", "-e", "-n", "-fast", "-y", "-delete", "-delete+", "-sensen"),
	}
}
