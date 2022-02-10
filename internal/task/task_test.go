// © Ben Garrett https://github.com/bengarrett/dupers
package task_test

import (
	"testing"

	"github.com/bengarrett/dupers/dupe"
	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/bengarrett/dupers/internal/task"
)

func TestHelp(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		if got := task.Help(); got == "" {
			t.Error("Help() returned nothing")
		}
	})
}

func TestDupe(t *testing.T) {
	type args struct {
		c    *dupe.Config
		f    *cmd.Flags
		args []string
	}
	var (
		c dupe.Config
		f cmd.Flags
	)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"nil", args{}, true},
		{"empty", args{&c, &f, []string{}}, true},
		{"test", args{&c, &f, []string{"test", "test"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := task.Dupe(tt.args.c, tt.args.f, tt.args.args...); (err != nil) != tt.wantErr {
				t.Errorf("Dupe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
