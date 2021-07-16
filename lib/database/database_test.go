package database

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gookit/color"
)

const (
	fileSrc  = "../../test/files_to_check/corrupt_program.exe"
	fileDest = "../../test/tmp/corrupt_program.exe"
)

func Test_copyFile(t *testing.T) {
	type args struct {
		src  string
		dest string
	}
	d, err := filepath.Abs(fileDest)
	if err != nil {
		t.Error(err)
	}
	s, err := filepath.Abs(fileSrc)
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{"empty", args{}, 0, true},
		{"exe", args{src: s, dest: d}, 10240, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := copyFile(tt.args.src, tt.args.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("copyFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuckets(t *testing.T) {
	color.Enable = false
	testMode = true
	tests := []struct {
		name      string
		wantNames []string
		wantErr   bool
	}{
		{"test", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNames, err := Buckets()
			if (err != nil) != tt.wantErr {
				t.Errorf("Buckets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNames, tt.wantNames) {
				t.Errorf("Buckets() = %v, want %v", gotNames, tt.wantNames)
			}
		})
	}
}
