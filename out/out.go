// © Ben Garrett https://github.com/bengarrett/dupers

package out

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/color"
)

// Bug prints the string to a newline.
func Bug(debug string) {
	fmt.Printf("∙%s\n", debug)
}

// ErrFatal prints the error and exits the program.
func ErrFatal(err error) {
	if err != nil {
		color.Error.Tips(" " + err.Error())
	}
	os.Exit(1)
}

// ErrFatal prints the error.
func ErrCont(err error) {
	if err == nil {
		return
	}
	color.Warn.Printf("The %s.\n", err.Error())
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
