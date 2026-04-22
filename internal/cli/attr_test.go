package cli

import (
	"strings"
	"testing"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
)

func TestAttrSet_Get_Delete(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "myentry", Group: "grp", Content: "data"})
	_ = db.Save()

	// set
	_, err := runCmd(t, "attr", "set", "--db", path, "--group", "grp", "myentry", "env=production")
	if err != nil {
		t.Fatalf("attr set: %v", err)
	}

	// verify via db
	db2, _ := kdbx.OpenDB(path, "pw")
	e, _ := db2.GetEntry("myentry", "grp")
	if e.Attributes["env"] != "production" {
		t.Errorf("expected env=production, got %q", e.Attributes["env"])
	}

	// get
	out, err := runCmd(t, "attr", "get", "--db", path, "--group", "grp", "myentry")
	if err != nil {
		t.Fatalf("attr get: %v", err)
	}
	if !strings.Contains(out, "production") {
		t.Errorf("expected 'production' in attr get output: %s", out)
	}

	// delete
	_, err = runCmd(t, "attr", "delete", "--db", path, "--group", "grp", "myentry", "env")
	if err != nil {
		t.Fatalf("attr delete: %v", err)
	}

	db3, _ := kdbx.OpenDB(path, "pw")
	e3, _ := db3.GetEntry("myentry", "grp")
	if _, ok := e3.Attributes["env"]; ok {
		t.Error("expected env attribute to be deleted")
	}
}

func TestAttrSet_InvalidPair(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	db, _ := kdbx.OpenDB(path, "pw")
	_ = db.AddEntry(kdbx.Entry{Title: "e", Group: "g", Content: "c"})
	_ = db.Save()

	_, err := runCmd(t, "attr", "set", "--db", path, "--group", "g", "e", "nogequals")
	if err == nil {
		t.Fatal("expected error for missing '=' in key=value pair")
	}
}
