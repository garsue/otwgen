package generate

import (
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
		if len(got[0].Syntax) != 1 {
			t.Errorf("LoadPackages() len(got[0].Syntax) = %d, want 1", len(got[0].Syntax))
		}
	})
}
