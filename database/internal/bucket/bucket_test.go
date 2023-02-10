// © Ben Garrett https://github.com/bengarrett/dupers
package bucket_test

import (
	"os"
	"testing"

	"github.com/bengarrett/dupers/database/internal/bucket"
	"github.com/bengarrett/dupers/internal/mock"
	bolt "go.etcd.io/bbolt"
)

func TestCleaner_Clean(t *testing.T) {
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
		return
	}
	db, err := mock.TestDB()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	type fields struct {
		DB    *bolt.DB
		Name  string
		Debug bool
		Quiet bool
		Items int
		Total int
		Finds int
		Errs  int
	}
	tests := []struct {
		name       string
		fields     fields
		wantCount  bool
		wantFinds  int
		wantErrors int
	}{
		{"empty", fields{}, false, 0, 0},
		{"defaults", fields{DB: db}, false, 0, 1},
		{"okay", fields{DB: db, Name: mock.Bucket1()}, true, 0, 0},
		{"debug", fields{DB: db, Name: mock.Bucket1(), Debug: true}, true, 0, 0},
		{"quiet", fields{DB: db, Name: mock.Bucket1(), Quiet: true}, true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &bucket.Cleaner{
				Name:  tt.fields.Name,
				Debug: tt.fields.Debug,
				Quiet: tt.fields.Quiet,
				Items: tt.fields.Items,
				Total: tt.fields.Total,
				Finds: tt.fields.Finds,
				Errs:  tt.fields.Errs,
			}
			gotCount, gotFinds, gotErrors, _ := c.Clean(tt.fields.DB)
			if (gotCount > 0) != tt.wantCount {
				t.Errorf("Cleaner.Clean() gotCount = %v, want %v", gotCount, tt.wantCount)
			}
			if gotFinds != tt.wantFinds {
				t.Errorf("Cleaner.Clean() gotFinds = %v, want %v", gotFinds, tt.wantFinds)
			}
			if gotErrors != tt.wantErrors {
				t.Errorf("Cleaner.Clean() gotErrors = %v, want %v", gotErrors, tt.wantErrors)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	t.Run("no blank", func(t *testing.T) {
		const wantErr = false
		got, err := bucket.Abs("test")
		if (err != nil) != wantErr {
			t.Errorf("Abs() error = %v, wantErr %v", err, wantErr)
			return
		}

		if got == "" {
			t.Error("Abs() returned an empty path")
		}
	})
}

func TestCount(t *testing.T) {
	if err := mock.TestRemove(); err != nil {
		t.Error(err)
		return
	}
	if err := mock.TestOpen(); err != nil {
		t.Error(err)
		return
	}
	db, err := mock.TestDB()
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()
	type args struct {
		name string
		db   *bolt.DB
	}
	tests := []struct {
		name      string
		args      args
		wantItems bool
		wantErr   bool
	}{
		{"empty", args{}, false, true},
		{"no bucket", args{db: db}, false, true},
		{"bad bucket", args{"abc", db}, false, true},
		{"404", args{mock.Bucket2(), db}, false, true},
		{"okay", args{mock.Bucket1(), db}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotItems, err := bucket.Count(tt.args.db, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Count() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (gotItems > 0) != tt.wantItems {
				t.Errorf("Count() = %v, want %v", gotItems, tt.wantItems)
			}
		})
	}
}

func TestStat(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name      string
		args      args
		wantEmpty bool
	}{
		{"empty", args{}, true},
		{"not found", args{"asdffdsaasdfdfa"}, true},
		{"temp", args{os.TempDir()}, false},
		{"current", args{"."}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bucket.Stat(tt.args.name, false, true); (got == "") != tt.wantEmpty {
				t.Errorf("Stat() = %v, want empty %v", got, tt.wantEmpty)
			}
		})
	}
}

func TestTotal(t *testing.T) {

	db, err := mock.TestDB()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	type args struct {
		buckets []string
		db      *bolt.DB
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"empty", args{}, 0, true},
		{"no buckets", args{db: db}, 0, false},
		{"bad buckets", args{[]string{"abc", "def"}, db}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bucket.Total(tt.args.db, tt.args.buckets)
			if (err != nil) != tt.wantErr {
				t.Errorf("Total() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Total() = %v, want %v", got, tt.want)
			}
		})
	}
}
