// © Ben Garrett https://github.com/bengarrett/dupers
package duplicate_test

import (
	"testing"

	"github.com/bengarrett/dupers/internal/task/internal/duplicate"
)

func TestCmdErr(t *testing.T) {
	type args struct {
		args    int
		buckets int
		minArgs int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"zero", args{args: 0, buckets: 0, minArgs: 0}, ""},
		{"one", args{args: 0, buckets: 0, minArgs: 1}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := duplicate.CmdErr(tt.args.args, tt.args.buckets, tt.args.minArgs); got != tt.want {
				t.Errorf("CmdErr() = %v, want %v", got, tt.want)
			}
		})
	}
}
