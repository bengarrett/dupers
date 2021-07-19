// © Ben Garrett https://github.com/bengarrett/dupers

package out

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/color"
)

func ErrFatal(e error) {
	if e != nil {
		color.Error.Tips(" " + e.Error())
	}
	os.Exit(1)
}

func ErrCont(e error) {
	if e == nil {
		return
	}
	color.Warn.Printf("The %s.\n", e.Error())
}

func Example(s string) {
	if s == "" {
		return
	}
	color.Debug.Println(s)
}

func Response(s string, quiet bool) {
	if quiet {
		return
	}
	fmt.Println(s)
}

func YN(s string) bool {
	fmt.Println()
	r := bufio.NewReader(os.Stdin)
	const no, yes = "n", "y"
	for {
		fmt.Printf("\r%s? [Y/N]: ", s)
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
