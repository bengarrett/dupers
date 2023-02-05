// © Ben Garrett https://github.com/bengarrett/dupers
package task_test

import (
	"testing"

	"github.com/bengarrett/dupers/database"
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
	const tester = true
	database.TestMode = tester
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
			if err := task.Dupe(tt.args.c, tt.args.f, tester, tt.args.args...); (err != nil) != tt.wantErr {
				t.Errorf("Dupe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabase(t *testing.T) {
	type args struct {
		c     *dupe.Config
		quiet bool
		cmd   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{}, true},
		{"invalid cmd", args{cmd: "xyz"}, true},
		{"backup", args{cmd: "backup"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := task.Database(tt.args.c, tt.args.quiet, tt.args.cmd); (err != nil) != tt.wantErr {
				t.Errorf("Database() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	// do not test, as it will use the users actual database
}
