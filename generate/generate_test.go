package generate

import (
	"bytes"
	"go/format"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"golang.org/x/tools/go/packages"
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
		if len(got[0].Syntax) != 3 {
			t.Errorf("LoadPackages() len(got[0].Syntax) = %d, want 1", len(got[0].Syntax))
		}
	})
}

// nolint: gocyclo
func TestNewFile(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedSyntax |
			packages.NeedTypes,
	}, "github.com/garsue/otwgen/generate/testdata")
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		syntax SyntaxTree
	}
	tests := []struct {
		name  string
		args  args
		want1 bool
	}{
		{
			name: "empty",
			args: args{
				syntax: SyntaxTree{
					pkg:  pkgs[0],
					file: pkgs[0].Syntax[0],
				},
			},
			want1: false,
		},
		{
			name: "func",
			args: args{
				syntax: SyntaxTree{
					pkg:  pkgs[0],
					file: pkgs[0].Syntax[1],
				},
			},
			want1: true,
		},
		{
			name: "struct",
			args: args{
				syntax: SyntaxTree{
					pkg:  pkgs[0],
					file: pkgs[0].Syntax[2],
				},
			},
			want1: true,
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := NewFile(tt.args.syntax)
			if got1 != tt.want1 {
				t.Errorf("NewFile() got1 = %v, want %v", got1, tt.want1)
			}
			if !got1 {
				return
			}
			buffer := bytes.NewBuffer(make([]byte, 0, 1024))
			if err1 := format.Node(buffer, token.NewFileSet(), got); err1 != nil {
				t.Error(err1)
				return
			}
			want, err := ioutil.ReadFile("testdata/" + tt.name + ".txt")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buffer.Bytes(), want) {
				t.Error(buffer.String())
				file, err := os.Create(tt.name + "-actual.txt")
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
		})
	}
}
