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
type YND uint

const (
	// Nil does not set a default value.
	Nil YND = iota
	// Yes sets the default value.
	Yes
	// No sets the default value.
	No
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
	// Read returns Reading items.
	Read

	// ANSI control code to erase the active terminal line.
	EraseLine = "\u001b[2K"
	cr        = "\r"
	winOS     = "windows"
)

// Bug prints the string to a newline.
func Bug(debug string) {
	fmt.Fprintf(os.Stderr, "∙%s\n", debug)
}

// EnterKey returns the Enter keyboard code.
func EnterKey() byte {
	const lf, cr = '\u000A', '\u000D'
	if runtime.GOOS == winOS {
		return cr
	}
	return lf
}

// ErrAppend prints the error to an active line.
func ErrAppend(err error) {
	if err == nil {
		return
	}
	s := strings.ToLower(err.Error())
	fmt.Fprint(os.Stderr, color.Warn.Sprintf("%s.\n", strings.TrimSpace(s)))
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
	return fmt.Sprintf("%s%s", EraseLine, cr)
}

// Status prints out the current file or item processing count.
func Status(count, total int, m Mode) string {
	// to significantly improved terminal performance
	// only update the status every 1000th count
	const mod, ten = 1000, 10
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
		pre = EraseLine + pre
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

// YN prints the question to stdout and prompts for a yes or no reply.
// The prompt will loop unless a y or n value is given or Ctrl-C is pressed.
func YN(question string, recommend YND) bool {
	const no, yes, cursorUp = "n", "y", "\x1b[1A"
	p, def := ynDefine(recommend)
	prompt := fmt.Sprintf("\r%s?%s[%s]: ", question, def, p)
	fmt.Print(prompt)
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
				fmt.Printf("%s%s%s\n", cursorUp, prompt, "y")
				return true
			case No:
				fmt.Printf("%s%s%s\n", cursorUp, prompt, "n")
				return false
			case Nil:
				continue
			}
		}
	}
}

func ynDefine(recommend YND) (p string, def string) {
	p, def = "", " "
	switch recommend {
	case Nil:
		p = "Y/N"
	case Yes:
		p = "Y/n"
		def = " (default: yes) "
	case No:
		p = "N/y"
		def = " (default: no) "
	}
	return p, def
}

// Prompt prints the question to stdout and prompts for a string reply.
// The prompt will loop until Enter key or Ctrl-C are pressed.
func Prompt(question string) string {
	r := bufio.NewReader(os.Stdin)
	fmt.Printf("\r%s?: ", question)
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
