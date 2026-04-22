package cli

import "testing"

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Error("Version() must not be empty")
	}
}
