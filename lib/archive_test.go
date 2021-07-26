// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package dupers

import (
	"log"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/mock"
)

func Test_extension(t *testing.T) {
	const xz = ".xz"
	tests := []struct {
		name string
		find string
		want string
	}{
		{"empty", "", ""},
		{"xz1", xz, appXZ},
		{"xz2", appXZ, xz},
		{"caps", strings.ToUpper(xz), appXZ},
		{"no dot", "xz", xz},
	}
	if err := mock.DBUp(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extension(tt.find); got != tt.want {
				t.Errorf("extension() = %v, want %v", got, tt.want)
			}
		})
	}
	if err := mock.DBDown(); err != nil {
		log.Fatal(err)
	}
}

func TestIsArchive(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantResult bool
		wantMime   string
		wantErr    bool
	}{
		{"empty", "", false, "", true},
		{"text file", "../test/randomfiles.txt", false, "", false},
		{"7z", "../test/randomfiles.7z", true, app7z, false},
		{"xz", "../test/randomfiles.tar.xz", false, appXZ, false},
		{"zip", "../test/randomfiles.zip", true, appZip, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotMime, err := IsArchive(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsArchive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResult != tt.wantResult {
				t.Errorf("IsArchive() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotMime != tt.wantMime {
				t.Errorf("IsArchive() gotMime = %v, want %v", gotMime, tt.wantMime)
			}
		})
	}
}

func TestIsExtension(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantResult bool
		wantMime   string
	}{
		{"empty", "", false, ""},
		{"text", "file.txt", false, ""},
		{"zip", "file.zip", true, appZip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotMime := IsExtension(tt.filename)
			if gotResult != tt.wantResult {
				t.Errorf("IsExtension() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotMime != tt.wantMime {
				t.Errorf("IsExtension() gotMime = %v, want %v", gotMime, tt.wantMime)
			}
		})
	}
}

func TestConfig_WalkArchiver(t *testing.T) {
	type fields struct {
		Debug    bool
		Quiet    bool
		Test     bool
		internal internal
	}
	type args struct {
		name Bucket
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"empty", fields{}, args{""}, true},
		{"bucket1", fields{Test: true}, args{Bucket(mock.Bucket1())}, false},
	}
	if err := mock.DBUp(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Debug:    tt.fields.Debug,
				Quiet:    tt.fields.Quiet,
				Test:     tt.fields.Test,
				internal: tt.fields.internal,
			}
			if err := c.WalkArchiver(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Config.WalkArchiver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	if err := mock.DBDown(); err != nil {
		log.Fatal(err)
	}
}

func TestConfig_read7Zip(t *testing.T) {
	type fields struct {
		Debug    bool
		Quiet    bool
		Test     bool
		internal internal
	}
	type args struct {
		bucket string
		name   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"empty", fields{Test: true}, args{}},
		{"7z", fields{Test: true}, args{mock.Bucket1(), mock.SevenZip}},
	}
	if err := mock.DBUp(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Debug:    tt.fields.Debug,
				Quiet:    tt.fields.Quiet,
				Test:     tt.fields.Test,
				internal: tt.fields.internal,
			}
			c.read7Zip(tt.args.bucket, tt.args.name)
		})
	}
	if err := mock.DBDown(); err != nil {
		log.Fatal(err)
	}
}
