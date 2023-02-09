// © Ben Garrett https://github.com/bengarrett/dupers
package cmd_test

import (
	"log"
	"testing"

	"github.com/bengarrett/dupers/internal/cmd"
	"github.com/gookit/color"
)

func TestChkWinDir(t *testing.T) {
	err := cmd.WindowsChk("")
	if err != nil {
		t.Errorf("WindowsChk empty should return nil, got %v", err)
	}
	err = cmd.WindowsChk("\"\"")
	if err != nil {
		t.Errorf("WindowsChk empty quotes should return nil, got %v", err)
	}
	err = cmd.WindowsChk("\"E:\"")
	if err != nil {
		t.Errorf("WindowsChk drive letter should return nil, got %v", err)
	}
	const letter = "C:\""
	err = cmd.WindowsChk(letter)
	if err == nil {
		t.Errorf("WindowsChk drive letter with backslash should return an error")
	}
	const path = "C:\\My Files"
	err = cmd.WindowsChk(path)
	if err != nil {
		t.Errorf("WindowsChk path should return nil, got %v", err)
	}
	const cmdPath = "C:\\My Files\\"
	err = cmd.WindowsChk(cmdPath)
	if err != nil {
		t.Errorf("WindowsChk path should return nil, got %v", err)
	}
}

func TestDefine(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("cannot run multiple counts of test:", err)
		}
	}()
	f := cmd.Flags{}
	f.Define()
	if *f.Version != false {
		t.Errorf("Define() version error = %v, want false", *f.Version)
	}
}

func TestDefineShort(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("cannot run multiple counts of test:", err)
		}
	}()
	a := cmd.Aliases{}
	a.Define()
	if *a.Version != false {
		t.Errorf("DefineShort() version error = %v, want false", *a.Version)
	}
}

func TestHome(t *testing.T) {
	t.Run("home", func(t *testing.T) {
		if cmd.Home() == "" {
			t.Error("Home() = \"\", want a directory path")
		}
	})
}

func TestSelf(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"expected", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cmd.Self()
			if (err != nil) != tt.wantErr {
				t.Errorf("Self() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestSearchSummary(t *testing.T) {
	type args struct {
		total    int
		term     string
		exact    bool
		filename bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{}, "No results exist for ''."},
		{"single", args{total: 1}, "1 result exist for ''."},
		{"two", args{total: 2}, "2 results exist for ''."},
		{"multiple results", args{term: "xyz", total: 3}, "3 results exist for 'xyz'."},
		{"exact results", args{term: "xyz", total: 3, exact: true}, "3 exact results exist for 'xyz'."},
		{"filename results", args{term: "xyz", total: 3, filename: true}, "3 filename results exist for 'xyz'."},
		{
			"exact filename results",
			args{term: "xyz", total: 3, exact: true, filename: true},
			"3 exact filename results exist for 'xyz'.",
		},
	}
	color.Enable = false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmd.SearchSummary(tt.args.total, tt.args.term, tt.args.exact, tt.args.filename); got != tt.want {
				t.Errorf("SearchSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}
