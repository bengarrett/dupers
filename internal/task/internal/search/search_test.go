// © Ben Garrett https://github.com/bengarrett/dupers
package search_test

import (
	"testing"

	"github.com/bengarrett/dupers/internal/task/internal/search"
	"github.com/bengarrett/dupers/pkg/database"
	bolt "go.etcd.io/bbolt"
)

func TestCmdErr(t *testing.T) {
	tests := []struct {
		name string
		l    int
	}{
		{"0", 0},
		{"2", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = search.CmdErr(tt.l, true)
		})
	}
}

func TestErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"empty", database.ErrEmpty},
		{"not found", bolt.ErrBucketNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = search.Error(tt.err, true)
		})
	}
}
