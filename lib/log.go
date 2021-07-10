// © Ben Garrett https://github.com/bengarrett/dupers

package dupers

import (
	"github.com/gookit/color"
)

// Error saves the error to either a new or append an existing log file.
func (c *Config) Error(err error) {
	if err == nil {
		return
	}
	color.Error.Tips(" " + err.Error())
	//c.WriteLog(fmt.Sprintf("ERROR: %s", err))
}
