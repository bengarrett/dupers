// © Ben Garrett https://github.com/bengarrett/dupers
package printer_test

import (
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/printer"
	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	s := ""
	s = printer.Status(-1, -1, 0)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(printer.Status(0, 0, 0), printer.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(printer.Status(1, 1000, 0), printer.Eraser)
	assert.Equal(t, "\rChecking 1 of 1,000 items ", s)
	s, _ = strings.CutPrefix(printer.Status(100, 100, 0), printer.Eraser)
	assert.Equal(t, "\rChecking 100 of 100 items ", s)
	s, _ = strings.CutPrefix(printer.Status(1000, 20000, printer.Check), printer.Eraser)
	assert.Equal(t, "\rChecking 1,000+ of 20,000 items ", s)
	s, _ = strings.CutPrefix(printer.Status(1001, 20000, printer.Check), printer.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(printer.Status(10001, 20000, printer.Check), printer.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(printer.Status(5000, 20000, printer.Look), printer.Eraser)
	assert.Equal(t, "\rLooking up 5,000+ items     ", s)
	s, _ = strings.CutPrefix(printer.Status(5000, 20000, printer.Scan), printer.Eraser)
	assert.Equal(t, "\rScanning 5,000+ files       ", s)
	s, _ = strings.CutPrefix(printer.Status(5000, 20000, printer.Read), printer.Eraser)
	assert.Equal(t, "\rReading 5,000+ of 20,000 items  ", s)
	s, _ = strings.CutPrefix(printer.Status(5000, 20000, 5), printer.Eraser)
	assert.Equal(t, "", s)
}
