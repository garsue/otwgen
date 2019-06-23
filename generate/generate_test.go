package generate

import (
	"bytes"
	"go/format"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadPackages(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		got, err := LoadPackages([]string{"github.com/garsue/otwgen/generate/testdata/foo"})
		if err != nil {
			t.Errorf("LoadPackages() error = %v, wantErr nil", err)
			return
		}
		if len(got) != 1 {
			t.Errorf("LoadPackages() len(got) = %d, want 1", len(got))
		}
		if len(got[0].Errors) != 1 {
			t.Errorf("LoadPackages() len(got[0].Errors) = %d, want 1", len(got[0].Errors))
		}
	})
	t.Run("found", func(t *testing.T) {
		got, err := LoadPackages([]string{"github.com/garsue/otwgen/generate/testdata"})
		if err != nil {
			t.Errorf("LoadPackages() error = %v, wantErr nil", err)
			return
		}
		if len(got) != 1 {
			t.Errorf("LoadPackages() len(got) = %d, want 1", len(got))
		}
		if got[0].Name != "testdata" {
			t.Errorf("LoadPackages() got[0].Name = %s, want %s", got[0].Name, "testdata")
		}
		if len(got[0].Syntax) != 2 {
			t.Errorf("LoadPackages() len(got[0].Syntax) = %d, want 1", len(got[0].Syntax))
		}
	})
}

func TestNewFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pkgs, err := LoadPackages([]string{"github.com/garsue/otwgen/generate/testdata"})
		if err != nil {
			t.Errorf("LoadPackages() error = %v, wantErr nil", err)
			return
		}
		got, got1 := NewFile(pkgs[0])
		buffer := bytes.NewBuffer(make([]byte, 0, 1024))
		if err1 := format.Node(buffer, token.NewFileSet(), got); err1 != nil {
			t.Error(err1)
			return
		}
		want, err := ioutil.ReadFile("testdata/gen.txt")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buffer.Bytes(), want) {
			t.Error(buffer.String())
			file, err := os.Create("actual.txt")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			if _, err := io.Copy(file, buffer); err != nil {
				t.Fatal(err)
			}
		}
		if !got1 {
			t.Errorf("NewFile() got1 = %v, want %v", got1, true)
		}
	})
}
