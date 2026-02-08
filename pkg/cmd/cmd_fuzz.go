// © Ben Garrett https://github.com/bengarrett/dupers

//go:build go1.18

package cmd

import (
	"testing"
)

// FuzzWindowsChk fuzz tests the Windows directory path validation.
func FuzzWindowsChk(f *testing.F) {
	// Add some initial test cases
	testCases := []string{
		`C:\path\to\dir`,
		`"C:\path\to\dir"`,
		`"C:\path\to\dir\"`,
		`/unix/path`,
		`"`,
		`simple`,
		`"C:\Users\Ben\Documents\"`,
		`C:\Users\Ben\Documents\`,
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, path string) {
		// This should not panic
		err := WindowsChk(path)

		// We expect either nil or an error, but not a panic
		// The function should handle all input gracefully
		_ = err
	})
}

// FuzzSearchSummary fuzz tests the search summary formatting.
func FuzzSearchSummary(f *testing.F) {
	// Add some initial test cases
	testCases := []struct {
		total    int
		term     string
		exact    bool
		filename bool
	}{
		{0, "search", false, false},
		{1, "file.txt", true, false},
		{100, "query", false, true},
		{1, "special!@#", true, true},
		{0, "", false, false},
		{1000, "unicode📁search", false, false},
	}

	for _, tc := range testCases {
		f.Add(tc.total, tc.term, tc.exact, tc.filename)
	}

	f.Fuzz(func(t *testing.T, total int, term string, exact bool, filename bool) {
		// This should not panic
		result := SearchSummary(total, term, exact, filename)

		// Result should be a non-empty string for valid inputs
		if total >= 0 {
			if result == "" {
				t.Fatal("Search summary should not be empty for non-negative total")
			}
		}
	})
}
