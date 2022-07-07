// © Ben Garrett https://github.com/bengarrett/dupers

// Mock is a set of simulated database and bucket functions for unit testing.
package mock_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/internal/mock"
)

func TestRootDir(t *testing.T) {
	got := mock.RootDir()
	_, err := os.Stat(got)
	if os.IsNotExist(err) {
		t.Errorf("RootDir() does not exist, %v", got)
	} else if err != nil {
		t.Errorf("RootDir() stat error = %v, want nil", err)
	}
}

func TestBucket2(t *testing.T) {
	s, err := mock.Bucket2()
	if err != nil {
		t.Errorf("Bucket2() error = %v, want nil", err)
	} else if s == "" {
		t.Errorf("Bucket2() s is empty")
	}
}

func TestExport1(t *testing.T) {
	got, err := mock.Export1()
	if err != nil {
		t.Errorf("Export1() error = %v, want nil", err)
	}
	if _, err := os.Stat(got); os.IsNotExist(err) {
		t.Errorf("Export1() file does not exist, %v", got)
	} else if err != nil {
		t.Errorf("Export1() stat error = %v, want nil", err)
	}
}

func TestOpenAdd(t *testing.T) {
	t.Run("create and open", func(t *testing.T) {
		db, err := mock.Open()
		if err != nil {
			t.Errorf("TestOpenAdd() error = %v, wantErr %v", err, nil)
		} else if db == nil {
			t.Errorf("TestOpenAdd() db = %v, want %v", nil, "bolt database")
		}
		defer db.Close()
		b1, err := mock.Bucket1()
		if err != nil {
			t.Errorf("TestOpenAdd() bucket1 error = %v, wantErr %v", err, nil)
		}
		i1, err := mock.Item1()
		if err != nil {
			t.Errorf("TestOpenAdd() item1 = %v, wantErr %v", err, nil)
		}
		if err := mock.CreateItem(b1, i1, db); err != nil {
			t.Errorf("TestOpenAdd() CreateItem error = %v, wantErr %v", err, nil)
		}
	})
}

func TestOpenRemove(t *testing.T) {
	t.Run("create and open", func(t *testing.T) {
		if err := mock.TestOpen(); err != nil {
			t.Errorf("TestOpen() error = %v, wantErr %v", err, nil)
		}
		defer func() {
			t.Run("remove", func(t *testing.T) {
				if err := mock.TestRemove(); err != nil {
					t.Errorf("TestRemove() error = %v, wantErr %v", err, nil)
				}
			})
		}()
	})
}
