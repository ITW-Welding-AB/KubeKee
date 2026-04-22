package kdbx

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDB(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.kdbx")
	if err := CreateDB(path, "testpassword"); err != nil {
		t.Fatalf("CreateDB: %v", err)
	}
	return path, func() { os.Remove(path) }
}

func TestCreateDB(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}

func TestOpenDB(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, err := OpenDB(path, "testpassword")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if db.Path != path {
		t.Errorf("expected path %q, got %q", path, db.Path)
	}
}

func TestOpenDB_WrongPassword(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	_, err := OpenDB(path, "wrongpassword")
	if err == nil {
		t.Fatal("expected error with wrong password, got nil")
	}
}

func TestAddEntry(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, err := OpenDB(path, "testpassword")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}

	entry := Entry{
		Title:     "my-secret",
		Group:     "production",
		Content:   "apiVersion: v1\nkind: Secret",
		Kind:      "Secret",
		Name:      "my-secret",
		Namespace: "default",
	}
	if err := db.AddEntry(entry); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	if err := db.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reopen and verify
	db2, err := OpenDB(path, "testpassword")
	if err != nil {
		t.Fatalf("OpenDB after save: %v", err)
	}
	entries := db2.ListEntries("production")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "my-secret" {
		t.Errorf("expected title %q, got %q", "my-secret", entries[0].Title)
	}
}

func TestAddEntry_Duplicate(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	e := Entry{Title: "dup", Group: "grp", Content: "x"}
	if err := db.AddEntry(e); err != nil {
		t.Fatalf("first AddEntry: %v", err)
	}
	if err := db.AddEntry(e); err == nil {
		t.Fatal("expected error on duplicate entry, got nil")
	}
}

func TestGetEntry(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	e := Entry{Title: "cfg", Group: "staging", Content: "data: foo", Kind: "ConfigMap"}
	_ = db.AddEntry(e)
	_ = db.Save()

	db2, _ := OpenDB(path, "testpassword")
	got, err := db2.GetEntry("cfg", "staging")
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if got.Title != "cfg" {
		t.Errorf("unexpected title: %q", got.Title)
	}
	if got.Kind != "ConfigMap" {
		t.Errorf("unexpected kind: %q", got.Kind)
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	_, err := db.GetEntry("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestUpdateEntry(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	e := Entry{Title: "upd", Group: "g", Content: "old"}
	_ = db.AddEntry(e)
	_ = db.Save()

	db2, _ := OpenDB(path, "testpassword")
	if err := db2.UpdateEntry("upd", "g", "new", nil); err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}
	_ = db2.Save()

	db3, _ := OpenDB(path, "testpassword")
	got, _ := db3.GetEntry("upd", "g")
	if got.Content != "new" {
		t.Errorf("expected content %q, got %q", "new", got.Content)
	}
}

func TestUpdateEntry_NotFound(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	err := db.UpdateEntry("ghost", "g", "x", nil)
	if err == nil {
		t.Fatal("expected error updating non-existent entry")
	}
}

func TestSetAndDeleteAttribute(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	e := Entry{Title: "att", Group: "g", Content: "y"}
	_ = db.AddEntry(e)

	if err := db.SetAttribute("att", "g", "env", "prod"); err != nil {
		t.Fatalf("SetAttribute: %v", err)
	}
	_ = db.Save()

	db2, _ := OpenDB(path, "testpassword")
	got, _ := db2.GetEntry("att", "g")
	if got.Attributes["env"] != "prod" {
		t.Errorf("expected attribute env=prod, got %q", got.Attributes["env"])
	}

	_ = db2.DeleteAttribute("att", "g", "env")
	_ = db2.Save()

	db3, _ := OpenDB(path, "testpassword")
	got3, _ := db3.GetEntry("att", "g")
	if _, ok := got3.Attributes["env"]; ok {
		t.Error("expected attribute env to be deleted")
	}
}

func TestDeleteAttribute_NotFound(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	e := Entry{Title: "att2", Group: "g", Content: "z"}
	_ = db.AddEntry(e)

	err := db.DeleteAttribute("att2", "g", "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting non-existent attribute")
	}
}

func TestListEntries_AllGroups(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	_ = db.AddEntry(Entry{Title: "a", Group: "g1", Content: "1"})
	_ = db.AddEntry(Entry{Title: "b", Group: "g2", Content: "2"})

	entries := db.ListEntries("")
	if len(entries) != 2 {
		t.Errorf("expected 2 entries across all groups, got %d", len(entries))
	}
}

func TestListEntries_FilteredGroup(t *testing.T) {
	path, cleanup := tempDB(t)
	defer cleanup()

	db, _ := OpenDB(path, "testpassword")
	_ = db.AddEntry(Entry{Title: "a", Group: "g1", Content: "1"})
	_ = db.AddEntry(Entry{Title: "b", Group: "g2", Content: "2"})

	entries := db.ListEntries("g1")
	if len(entries) != 1 || entries[0].Title != "a" {
		t.Errorf("expected 1 entry in g1, got %d", len(entries))
	}
}
