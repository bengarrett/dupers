// © Ben Garrett https://github.com/bengarrett/dupers
package out

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/gookit/color"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// YND is the YN default value to use when there is no input given.
type YN uint

const (
	Nil YN = iota // Nil does not set a default value.
	Yes           // Yes sets the default value.
	No            // No sets the default value.
)

// Mode for the current processing count.
type Mode uint

const (
	Check Mode = iota // Check returns Checking items.
	Look              // Look returns Looking up items.
	Scan              // Scan returns Scanning files.
	Read              // Read returns Reading items.
)

const (
	Eraser      = "\u001b[2K" // Erase is an ANSI control code to erase the current line in stdout.
	CursorUp    = "\x1b[1A"   // CursorUp is an ANSI control to move the cursor up one line.
	MatchPrefix = "\n  ⤷\t"   // MatchPrefix is the prefix that's applied to dupe matches.

	cr    = "\r"
	winOS = "windows"
)

// DPrint prints the string to a newline.
func DPrint(debug bool, s string) {
	if !debug {
		return
	}
	fmt.Fprintf(os.Stdout, "∙%s\n", s)
}

// EnterKey returns the Enter keyboard code.
func EnterKey() byte {
	const lf, cr = '\u000A', '\u000D'
	if runtime.GOOS == winOS {
		return cr
	}

	return lf
}

// Stderr formats and prints the err to stderr.
func Stderr(err error) {
	if err == nil {
		return
	}

	s := strings.ToLower(err.Error())
	fmt.Fprint(os.Stderr, color.Warn.Sprintf("%s.", strings.TrimSpace(s)))
	fmt.Fprintln(os.Stderr)
}

// StderrCR formats and prints the err to current line of  stderr.
func StderrCR(err error) {
	if err == nil {
		return
	}

	const nf = "bucket not found:"

	s := err.Error()

	switch {
	case strings.HasPrefix(s, nf):
		fmt.Fprintln(os.Stdout, color.Info.Sprintf("%s\n",
			strings.Replace(s, nf, "New database bucket:", 1)))
		return
	case strings.HasPrefix(s, "bucket not found"):
		s = "bucket does not exist"
	}

	fmt.Fprintln(os.Stderr, color.Warn.Sprintf("\rThe %s", s))
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

	fmt.Fprintln(os.Stdout, color.Debug.Sprint(cmd))
}

// Response prints the string when quiet is false.
func Response(s string, quiet bool) {
	if quiet {
		return
	}

	fmt.Fprintf(os.Stdout, "%s\n", s)
}

// EraseLine uses ANSI to erase the current line in stdout.
func EraseLine() string {
	if runtime.GOOS == winOS {
		return ""
	}

	return fmt.Sprintf("%s%s", Eraser, cr)
}

// Status prints out the current file or item processing count.
func Status(count, total int, m Mode) string {
	// to significantly improved terminal performance
	// only update the status every 1000th count
	const mod, ten = 1000, 10
	if count < 0 || total <= 0 {
		return ""
	}
	if count != total && count > mod {
		if count < mod*2 {
			// between 1000-2000, update every 100th count
			if count%(mod/ten) != 0 {
				return ""
			}
		} else if count%mod != 0 {
			// 2000+, update every 1000th count
			return ""
		}
	}

	var (
		check = "%sChecking %d of %d items "
		look  = "%sLooking up %d items     "
		scan  = "%sScanning %d files       "
		read  = "%sReading %d of %d items  "
	)

	skipping, updating := (count >= mod), (count != total)

	if skipping && updating {
		check = "%sChecking %d+ of %d items "
		look = "%sLooking up %d+ items     "
		scan = "%sScanning %d+ files       "
		read = "%sReading %d+ of %d items  "
	}

	pre, p := cr, message.NewPrinter(language.English)

	if runtime.GOOS != winOS {
		// erasing the line makes for a less flickering counter.
		// not all Windows terminals support ANSI controls.
		pre = Eraser + pre
	}

	switch m {
	case Check:
		return p.Sprintf(check, pre, number.Decimal(count), number.Decimal(total))
	case Look:
		return p.Sprintf(look, pre, number.Decimal(count))
	case Scan:
		return p.Sprintf(scan, pre, number.Decimal(count))
	case Read:
		return p.Sprintf(read, pre, number.Decimal(count), number.Decimal(total))
	}

	return ""
}

// AskYN prints the question to stdout and prompts for a yes or no reply.
// The prompt will loop unless a y or n value is given or Ctrl-C is pressed.
// alwaysYes will display the question but automatically input "y" onbehalf of the user.
func AskYN(question string, alwaysYes bool, recommend YN) bool {
	const no, yes = "n", "y"

	w := os.Stdout
	prompt, suffix := recommend.Define()
	ask := fmt.Sprintf("\r%s?%s[%s]: ", question, prompt, suffix)
	fmt.Fprintf(w, "%s", ask)
	if alwaysYes {
		fmt.Fprintln(w, yes)
		return true
	}
	for {
		r := bufio.NewReader(os.Stdin)
		b, err := r.ReadByte()
		if err != nil {
			ErrFatal(err)
		}

		switch strings.ToLower(string(b)) {
		case yes:
			return true
		case no:
			return false
		}

		if b == EnterKey() {
			switch recommend {
			case Yes:
				fmt.Fprintf(w, "%s%s%s\n", CursorUp, ask, "y")
				return true
			case No:
				fmt.Fprintf(w, "%s%s%s\n", CursorUp, ask, "n")
				return false
			case Nil:
				continue
			}
		}
	}
}

func (y YN) Define() (string, string) {
	prompt, suffix := "", ""
	switch y {
	case Nil:
		prompt = "Y/N"
	case Yes:
		prompt = "Y/n"
		suffix = " (default: yes) "
	case No:
		prompt = "N/y"
		suffix = " (default: no) "
	}
	return prompt, suffix
}

// Prompt prints the question to stdout and prompts for a string reply.
// The prompt will loop until Enter key or Ctrl-C are pressed.
func Prompt(question string) string {
	r := bufio.NewReader(os.Stdin)

	fmt.Fprintf(os.Stdout, "\r%s?: ", question)

	for {
		s, err := r.ReadString(EnterKey())
		if err != nil {
			ErrFatal(err)
		}

		if s != "" {
			// remove the Enter key newline from the string
			// as this character will break directory and filepaths
			return strings.TrimSuffix(s, "\n")
		}
	}
}
