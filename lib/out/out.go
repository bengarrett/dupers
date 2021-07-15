// © Ben Garrett https://github.com/bengarrett/dupers

package out

import (
	"fmt"
	"os"

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
