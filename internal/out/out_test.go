// © Ben Garrett https://github.com/bengarrett/dupers
package out_test

import (
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	s := ""
	s = out.Status(-1, -1, 0)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(out.Status(0, 0, 0), out.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(out.Status(1, 1000, 0), out.Eraser)
	assert.Equal(t, "\rChecking 1 of 1,000 items ", s)
	s, _ = strings.CutPrefix(out.Status(100, 100, 0), out.Eraser)
	assert.Equal(t, "\rChecking 100 of 100 items ", s)
	s, _ = strings.CutPrefix(out.Status(1000, 20000, out.Check), out.Eraser)
	assert.Equal(t, "\rChecking 1,000+ of 20,000 items ", s)
	s, _ = strings.CutPrefix(out.Status(1001, 20000, out.Check), out.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(out.Status(10001, 20000, out.Check), out.Eraser)
	assert.Equal(t, "", s)
	s, _ = strings.CutPrefix(out.Status(5000, 20000, out.Look), out.Eraser)
	assert.Equal(t, "\rLooking up 5,000+ items     ", s)
	s, _ = strings.CutPrefix(out.Status(5000, 20000, out.Scan), out.Eraser)
	assert.Equal(t, "\rScanning 5,000+ files       ", s)
	s, _ = strings.CutPrefix(out.Status(5000, 20000, out.Read), out.Eraser)
	assert.Equal(t, "\rReading 5,000+ of 20,000 items  ", s)
	s, _ = strings.CutPrefix(out.Status(5000, 20000, 5), out.Eraser)
	assert.Equal(t, "", s)
}
