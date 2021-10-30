// © Ben Garrett https://github.com/bengarrett/dupers

// Package dupers is the blazing-fast file duplicate checker and filename search.
package dupe

import (
	"log"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/mock"
)

func Test_extension(t *testing.T) {
	t.Parallel()
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
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extension(tt.find); got != tt.want {
				t.Errorf("extension() = %v, want %v", got, tt.want)
			}
		})
	}
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}

func TestIsArchive(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	type args struct {
		name Bucket
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{""}, true},
		{"bucket1", args{Bucket(mock.Bucket1())}, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error
			c := Config{
				Test: true,
			}
			c.db, err = mock.Open()
			if err != nil {
				t.Error(err)
			}
			defer c.db.Close()
			if err := c.WalkArchiver(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Config.WalkArchiver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_read7Zip(t *testing.T) {
	t.Parallel()
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
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Config{
				Debug:    tt.fields.Debug,
				Quiet:    tt.fields.Quiet,
				Test:     tt.fields.Test,
				internal: tt.fields.internal,
			}
			c.read7Zip(tt.args.bucket, tt.args.name)
		})
	}
	if err := mock.TestRemove(); err != nil {
		log.Fatal(err)
	}
}
