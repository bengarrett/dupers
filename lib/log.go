// © Ben Garrett https://github.com/bengarrett/dupers

package dupers

import (
	"fmt"

	"github.com/gookit/color"
)

// Error saves the error to either a new or append an existing log file.
func (c *Config) Error(err error) {
	if err == nil {
		return
	}
	color.Error.Tips(fmt.Sprint(err))
	//c.WriteLog(fmt.Sprintf("ERROR: %s", err))
}
