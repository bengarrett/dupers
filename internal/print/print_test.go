// © Ben Garrett https://github.com/bengarrett/dupers
package print_test

import (
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/print"
	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	s := ""
	s = print.Status(-1, -1, 0)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(print.Status(0, 0, 0), print.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(print.Status(1, 1000, 0), print.Eraser)
	assert.Equal(t, "\rChecking 1 of 1,000 items ", s)
	s, _ = strings.CutPrefix(print.Status(100, 100, 0), print.Eraser)
	assert.Equal(t, "\rChecking 100 of 100 items ", s)
	s, _ = strings.CutPrefix(print.Status(1000, 20000, print.Check), print.Eraser)
	assert.Equal(t, "\rChecking 1,000+ of 20,000 items ", s)
	s, _ = strings.CutPrefix(print.Status(1001, 20000, print.Check), print.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(print.Status(10001, 20000, print.Check), print.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(print.Status(5000, 20000, print.Look), print.Eraser)
	assert.Equal(t, "\rLooking up 5,000+ items     ", s)
	s, _ = strings.CutPrefix(print.Status(5000, 20000, print.Scan), print.Eraser)
	assert.Equal(t, "\rScanning 5,000+ files       ", s)
	s, _ = strings.CutPrefix(print.Status(5000, 20000, print.Read), print.Eraser)
	assert.Equal(t, "\rReading 5,000+ of 20,000 items  ", s)
	s, _ = strings.CutPrefix(print.Status(5000, 20000, 5), print.Eraser)
	assert.Equal(t, "", s)
}
