package operator

import (
	"testing"
	"time"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
)

func TestGetInterval_Valid(t *testing.T) {
	r := &KeePassSourceReconciler{}
	d := r.getInterval("10m")
	if d != 10*time.Minute {
		t.Errorf("expected 10m, got %v", d)
	}
}

func TestGetInterval_Empty(t *testing.T) {
	r := &KeePassSourceReconciler{}
	d := r.getInterval("")
	if d != 5*time.Minute {
		t.Errorf("expected default 5m, got %v", d)
	}
}

func TestGetInterval_Invalid(t *testing.T) {
	r := &KeePassSourceReconciler{}
	d := r.getInterval("notaduration")
	if d != 5*time.Minute {
		t.Errorf("expected default 5m for invalid, got %v", d)
	}
}

func TestFilterEntries_NoFilter(t *testing.T) {
	r := &KeePassSourceReconciler{}
	// Build an in-memory DB with entries using a temp file
	dir := t.TempDir()
	path := dir + "/test.kdbx"
	if err := kdbx.CreateDB(path, "pw"); err != nil {
		t.Fatal(err)
	}
	db, err := kdbx.OpenDB(path, "pw")
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AddEntry(kdbx.Entry{Title: "a", Group: "g1", Content: "1"})
	_ = db.AddEntry(kdbx.Entry{Title: "b", Group: "g2", Content: "2"})

	entries := r.filterEntries(db, nil, nil)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestFilterEntries_GroupFilter(t *testing.T) {
	r := &KeePassSourceReconciler{}
	dir := t.TempDir()
	path := dir + "/test.kdbx"
	_ = kdbx.CreateDB(path, "pw")
	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "a", Group: "g1", Content: "1"})
	_ = db.AddEntry(kdbx.Entry{Title: "b", Group: "g2", Content: "2"})

	entries := r.filterEntries(db, []string{"g1"}, nil)
	if len(entries) != 1 || entries[0].Title != "a" {
		t.Errorf("expected 1 entry from g1, got %d", len(entries))
	}
}

func TestFilterEntries_EntryFilter(t *testing.T) {
	r := &KeePassSourceReconciler{}
	dir := t.TempDir()
	path := dir + "/test.kdbx"
	_ = kdbx.CreateDB(path, "pw")
	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "alpha", Group: "g", Content: "1"})
	_ = db.AddEntry(kdbx.Entry{Title: "beta", Group: "g", Content: "2"})
	_ = db.AddEntry(kdbx.Entry{Title: "gamma", Group: "g", Content: "3"})

	entries := r.filterEntries(db, nil, []string{"alpha", "gamma"})
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	titles := map[string]bool{}
	for _, e := range entries {
		titles[e.Title] = true
	}
	if !titles["alpha"] || !titles["gamma"] {
		t.Error("expected alpha and gamma in results")
	}
}

func TestFilterEntries_GroupAndEntryFilter(t *testing.T) {
	r := &KeePassSourceReconciler{}
	dir := t.TempDir()
	path := dir + "/test.kdbx"
	_ = kdbx.CreateDB(path, "pw")
	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "x", Group: "g1", Content: "1"})
	_ = db.AddEntry(kdbx.Entry{Title: "y", Group: "g1", Content: "2"})
	_ = db.AddEntry(kdbx.Entry{Title: "z", Group: "g2", Content: "3"})

	entries := r.filterEntries(db, []string{"g1"}, []string{"x"})
	if len(entries) != 1 || entries[0].Title != "x" {
		t.Errorf("expected only entry x, got %d entries", len(entries))
	}
}
