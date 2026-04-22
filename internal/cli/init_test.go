package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.kdbx")
	t.Setenv("KUBEKEE_PASSWORD", "testpw")

	out, err := runCmd(t, "init", "--db", path)
	if err != nil {
		t.Fatalf("init cmd: %v\noutput: %s", err, out)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Error("expected db file to exist after init")
	}
}

func TestInitCmd_NoDb(t *testing.T) {
	t.Setenv("KUBEKEE_PASSWORD", "pw")
	_, err := runCmd(t, "init")
	if err == nil {
		t.Fatal("expected error when --db is not provided")
	}
}
