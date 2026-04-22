package cli

import (
	"strings"
	"testing"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
)

func TestListCmd_Empty(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	out, err := runCmd(t, "list", "--db", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No entries found") {
		t.Errorf("expected 'No entries found', got: %s", out)
	}
}

func TestListCmd_WithEntries(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "sec", Group: "prod", Kind: "Secret", Content: "x"})
	_ = db.Save()

	out, err := runCmd(t, "list", "--db", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sec") {
		t.Errorf("expected entry 'sec' in output, got: %s", out)
	}
	if !strings.Contains(out, "prod") {
		t.Errorf("expected group 'prod' in output, got: %s", out)
	}
}

func TestListCmd_GroupFilter(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "a", Group: "g1", Content: "1"})
	_ = db.AddEntry(kdbx.Entry{Title: "b", Group: "g2", Content: "2"})
	_ = db.Save()

	out, err := runCmd(t, "list", "--db", path, "--group", "g1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a") {
		t.Errorf("expected entry 'a' in output: %s", out)
	}
	if strings.Contains(out, "b") {
		t.Errorf("entry 'b' should not appear when filtering by g1: %s", out)
	}
}
