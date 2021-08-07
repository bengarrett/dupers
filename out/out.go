// © Ben Garrett https://github.com/bengarrett/dupers

package out

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/gookit/color"
)

// Mode for the current processing count.
type Mode uint

const (
	// Check returns Checking items.
	Check Mode = iota
	// Look returns Looking up items.
	Look
	// Scan returns Scanning files.
	Scan

	eraseLine = "\u001b[2K"
	cr        = "\r"
	winOS     = "windows"
)

// Bug prints the string to a newline.
func Bug(debug string) {
	fmt.Printf("∙%s\n", debug)
}

// ErrAppend prints the error to an active line.
func ErrAppend(err error) {
	if err == nil {
		return
	}
	s := strings.ToLower(err.Error())
	color.Warn.Printf("%s.\n", strings.TrimSpace(s))
}

// ErrCont prints the error.
func ErrCont(err error) {
	if err == nil {
		return
	}
	const nf = "bucket not found:"
	s := err.Error()
	switch {
	case strings.HasPrefix(s, nf):
		color.Info.Printf("%s\n",
			strings.Replace(s, nf, "New database bucket:", 1))
		return
	case strings.HasPrefix(s, "bucket not found"):
		s = "bucket does not exist"
	}
	color.Warn.Printf("The %s.\n", s)
}

// ErrFatal prints the error and exits the program.
func ErrFatal(err error) {
	if err != nil {
		color.Error.Tips(" " + err.Error())
	}
	os.Exit(1)
}

// Example is intended for help screens and prints the example command.
func Example(cmd string) {
	if cmd == "" {
		return
	}
	color.Debug.Println(cmd)
}

// Response prints the string when quiet is false.
func Response(s string, quiet bool) {
	if quiet {
		return
	}
	fmt.Println(s)
}

func RMLine() string {
	if runtime.GOOS == winOS {
		return ""
	}
	return fmt.Sprintf("%s%s", eraseLine, cr)
}

// Status prints out the current file or item processing count.
func Status(count, total int, m Mode) string {
	const (
		check = "%sChecking %d of %d items "
		look  = "%sLooking up %d items     "
		scan  = "%sScanning %d files       "
	)
	pre := cr
	if runtime.GOOS != winOS {
		// erasing the line makes for a less flickering counter.
		// not all Windows terminals support ANSI controls.
		pre = eraseLine + pre
	}
	switch m {
	case Check:
		return fmt.Sprintf(check, pre, count, total)
	case Look:
		return fmt.Sprintf(look, pre, count)
	case Scan:
		return fmt.Sprintf(scan, pre, count)
	}
	return ""
}

// YN prints the question and prompts the user for a yes or no reply.
// The prompt will loop unless a y or n value is given or Ctrl-C is pressed.
func YN(question string) bool {
	fmt.Println()
	r := bufio.NewReader(os.Stdin)
	const no, yes = "n", "y"
	for {
		fmt.Printf("\r%s? [Y/N]: ", question)
		b, err := r.ReadByte()
		if err != nil {
			ErrFatal(err)
		}
		input := strings.ToLower(string(b))
		switch input {
		case yes:
			return true
		case no:
			return false
		}
	}
}
