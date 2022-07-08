// © Ben Garrett https://github.com/bengarrett/dupers
package out_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bengarrett/dupers/internal/out"
	"github.com/gookit/color"
	cap "github.com/zenizh/go-capturer"
)

func TestDebugLn(t *testing.T) {
	const hi = "Hello world!"
	t.Run("enter", func(t *testing.T) {
		s := cap.CaptureStderr(func() {
			out.DebugLn(hi)
		})
		if s != fmt.Sprintf("∙%s\n", hi) {
			t.Errorf("DebugLn() did not return the expected stderr, got %v", s)
		}
	})
}

func TestEnterKey(t *testing.T) {
	const lf, cr = '\u000A', '\u000D'
	t.Run("enter", func(t *testing.T) {
		b := out.EnterKey()
		if bytes.Equal([]byte{b}, []byte{cr}) {
			return
		}
		if bytes.Equal([]byte{b}, []byte{lf}) {
			return
		}
		t.Errorf("EnterKey returned an invalid byte, %v", b)
	})
}

func TestErrAppend(t *testing.T) {
	ErrTest := errors.New("hello world")
	color.Enable = false
	t.Run("enter", func(t *testing.T) {
		out := cap.CaptureStderr(func() {
			out.ErrAppend(ErrTest)
		})
		if out != fmt.Sprintf("%s.\n", ErrTest.Error()) {
			t.Errorf("ErrAppend() did not return the expected stderr, got %q", out)
		}
	})
}

func TestErrCont(t *testing.T) {
	ErrTest := errors.New("hello world")
	ErrNew := errors.New("bucket not found: abc")
	ErrNF := errors.New("bucket not found")
	color.Enable = false
	t.Run("enter", func(t *testing.T) {
		out := cap.CaptureStderr(func() {
			out.ErrCont(ErrTest)
		})
		if out != fmt.Sprintf("\rThe %s\n", strings.ToLower(ErrTest.Error())) {
			t.Errorf("ErrCont() did not return the expected stderr, got %q", out)
		}
	})
	t.Run("new", func(t *testing.T) {
		out := cap.CaptureStdout(func() {
			out.ErrCont(ErrNew)
		})
		if out != "New database bucket: abc\n\n" {
			t.Errorf("ErrCont() did not return the expected stdout, got %q", out)
		}
	})
	t.Run("not found", func(t *testing.T) {
		out := cap.CaptureStderr(func() {
			out.ErrCont(ErrNF)
		})
		if out != "\rThe bucket does not exist\n" {
			t.Errorf("ErrCont() did not return the expected stderr, got %q", out)
		}
	})
}

func TestStatus(t *testing.T) {
	type args struct {
		count int
		total int
		m     out.Mode
	}
	tests := []struct {
		name    string
		args    args
		wantStr bool
		wantErr bool
	}{
		{"empty", args{}, true, false},
		{"invalid", args{10, 10, 5}, false, true},
		{"check", args{10, 10, out.Check}, true, false},
		{"look", args{10, 10, out.Look}, true, false},
		{"look mod", args{1101, 10000, out.Look}, false, false},
		{"scan", args{10, 10, out.Scan}, true, false},
		{"read", args{10, 10, out.Read}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := out.Status(tt.args.count, tt.args.total, tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("Status() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != "") != tt.wantStr {
				t.Errorf("Status() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

func TestErrTip(t *testing.T) {
	ErrTest := errors.New("hello world")
	t.Run("blank", func(t *testing.T) {
		if s := out.ErrTip(nil); s != "" {
			t.Errorf("ErrTip() did not return the expected blank string, got %q", s)
		}
	})
	t.Run("no ansi", func(t *testing.T) {
		color.Enable = false
		s := out.ErrTip(ErrTest)
		if s != "ERROR: hello world\n" {
			t.Errorf("ErrTip() did not return the expected string, got %q", s)
		}
	})
}

func TestExampleLn(t *testing.T) {
	cmd := "hello world"
	color.Enable = false
	t.Run("enter", func(t *testing.T) {
		out := cap.CaptureStdout(func() {
			out.ExampleLn(cmd)
		})
		if out != fmt.Sprintf("%s\n", cmd) {
			t.Errorf("ErrAppend() did not return the expected stdout, got %q", out)
		}
	})
}

func TestResponse(t *testing.T) {
	cmd := "hello world"
	color.Enable = false
	t.Run("enter", func(t *testing.T) {
		s := cap.CaptureStdout(func() {
			out.Response(cmd, false)
		})
		if s != fmt.Sprintf("%s\n", cmd) {
			t.Errorf("Response() did not return the expected stdout, got %q", s)
		}
		s = cap.CaptureStdout(func() {
			out.Response(cmd, true)
		})
		if s != "" {
			t.Errorf("Response() did not return the expected blank string, got %q", s)
		}
	})
}
