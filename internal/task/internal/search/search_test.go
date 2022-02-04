package search_test

import (
	"testing"

	"github.com/bengarrett/dupers/database"
	"github.com/bengarrett/dupers/internal/task/internal/search"
)

func TestSearchCmdErr(t *testing.T) {
	tests := []struct {
		name string
		l    int
	}{
		{"0", 0},
		{"2", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			search.CmdErr(tt.l, true)
		})
	}
}

func TestSearchErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"empty", database.ErrDBEmpty},
		{"not found", database.ErrBucketNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			search.Error(tt.err, true)
		})
	}
}
