package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
)

func TestImportCmd_YAML(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	yamlFile := filepath.Join(t.TempDir(), "secret.yaml")
	content := `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: production
data:
  key: dmFsdWU=
`
	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, "import", "--db", path, yamlFile)
	if err != nil {
		t.Fatalf("import cmd: %v\noutput: %s", err, out)
	}

	db, _ := kdbx.OpenDB(path, "pw")
	entries := db.ListEntries("")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "my-secret" {
		t.Errorf("unexpected title: %q", entries[0].Title)
	}
	if entries[0].Kind != "Secret" {
		t.Errorf("unexpected kind: %q", entries[0].Kind)
	}
}

func TestImportCmd_GroupOverride(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	yamlFile := filepath.Join(t.TempDir(), "cm.yaml")
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: staging
data:
  foo: bar
`
	_ = os.WriteFile(yamlFile, []byte(content), 0644)

	_, err := runCmd(t, "import", "--db", path, "--group", "override-group", yamlFile)
	if err != nil {
		t.Fatal(err)
	}

	db, _ := kdbx.OpenDB(path, "pw")
	entries := db.ListEntries("override-group")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry in override-group, got %d", len(entries))
	}
}

func TestImportCmd_MultipleFiles(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	dir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		body := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + strings.TrimSuffix(name, ".yaml") + "\n"
		_ = os.WriteFile(filepath.Join(dir, name), []byte(body), 0644)
	}

	_, err := runCmd(t, "import", "--db", path,
		filepath.Join(dir, "a.yaml"), filepath.Join(dir, "b.yaml"))
	if err != nil {
		t.Fatalf("multi-file import: %v", err)
	}

	db, _ := kdbx.OpenDB(path, "pw")
	if got := len(db.ListEntries("")); got != 2 {
		t.Errorf("expected 2 entries, got %d", got)
	}
}

func TestImportFile_YAML(t *testing.T) {
	path := newTestDB(t, "pw")
	db, _ := kdbx.OpenDB(path, "pw")

	f := filepath.Join(t.TempDir(), "cm.yaml")
	_ = os.WriteFile(f, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: my-cm
  namespace: dev
`), 0644)

	importGroup = ""
	if err := importFile(db, f, false); err != nil {
		t.Fatalf("importFile: %v", err)
	}

	entries := db.ListEntries("dev")
	if len(entries) != 1 || entries[0].Title != "my-cm" {
		t.Errorf("expected entry my-cm in group dev, got %+v", entries)
	}
}

func TestImportFile_JSON(t *testing.T) {
	path := newTestDB(t, "pw")
	db, _ := kdbx.OpenDB(path, "pw")

	f := filepath.Join(t.TempDir(), "sec.json")
	_ = os.WriteFile(f, []byte(`{"kind":"Secret","metadata":{"name":"json-secret","namespace":"prod"}}`), 0644)

	importGroup = ""
	if err := importFile(db, f, false); err != nil {
		t.Fatalf("importFile JSON: %v", err)
	}

	entries := db.ListEntries("prod")
	if len(entries) != 1 || entries[0].Title != "json-secret" {
		t.Errorf("unexpected entries: %+v", entries)
	}
}

func TestImportFile_RawUnknownExtension(t *testing.T) {
	path := newTestDB(t, "pw")
	db, _ := kdbx.OpenDB(path, "pw")

	f := filepath.Join(t.TempDir(), "data.txt")
	_ = os.WriteFile(f, []byte("some raw content"), 0644)

	importGroup = "raw-group"
	if err := importFile(db, f, false); err != nil {
		t.Fatalf("importFile raw: %v", err)
	}

	entries := db.ListEntries("raw-group")
	if len(entries) != 1 {
		t.Errorf("expected 1 raw entry, got %d", len(entries))
	}
}

func TestImportFile_Attributes(t *testing.T) {
	path := newTestDB(t, "pw")
	db, _ := kdbx.OpenDB(path, "pw")

	f := filepath.Join(t.TempDir(), "annotated.yaml")
	_ = os.WriteFile(f, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: annotated
  annotations:
    custom.io/owner: team-a
    kubekee.env: staging
`), 0644)

	importGroup = ""
	if err := importFile(db, f, false); err != nil {
		t.Fatalf("importFile with annotations: %v", err)
	}

	entries := db.ListEntries("")
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	e := entries[0]
	if e.Attributes["env"] != "staging" {
		t.Errorf("expected env=staging, got %q", e.Attributes["env"])
	}
	if e.Attributes["custom.io/owner"] != "team-a" {
		t.Errorf("expected custom.io/owner=team-a, got %q", e.Attributes["custom.io/owner"])
	}
	if e.Attributes["createdAt"] == "" {
		t.Error("expected createdAt attribute to be set")
	}
}

func TestImportFile_DuplicateWithoutForce(t *testing.T) {
	path := newTestDB(t, "pw")
	db, _ := kdbx.OpenDB(path, "pw")

	f := filepath.Join(t.TempDir(), "dup.yaml")
	_ = os.WriteFile(f, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: dup-entry
  namespace: ns
`), 0644)

	importGroup = ""
	if err := importFile(db, f, false); err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Second import without --force must return an error.
	err := importFile(db, f, false)
	if err == nil {
		t.Fatal("expected error on duplicate import without --force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestImportFile_DuplicateWithForce(t *testing.T) {
	path := newTestDB(t, "pw")
	db, _ := kdbx.OpenDB(path, "pw")

	dir := t.TempDir()
	f := filepath.Join(dir, "update.yaml")
	original := `apiVersion: v1
kind: ConfigMap
metadata:
  name: updatable
  namespace: ns
data:
  v: "1"
`
	_ = os.WriteFile(f, []byte(original), 0644)

	importGroup = ""
	if err := importFile(db, f, false); err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Overwrite file with new content and re-import with force.
	updated := `apiVersion: v1
kind: ConfigMap
metadata:
  name: updatable
  namespace: ns
data:
  v: "2"
`
	_ = os.WriteFile(f, []byte(updated), 0644)

	if err := importFile(db, f, true); err != nil {
		t.Fatalf("force import: %v", err)
	}

	entries := db.ListEntries("ns")
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after force import, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Content, `v: "2"`) {
		t.Errorf("expected updated content, got:\n%s", entries[0].Content)
	}
}

func TestImportCmd_ForceFlag(t *testing.T) {
	path := newTestDB(t, "pw")
	t.Setenv("KUBEKEE_PASSWORD", "pw")

	yamlFile := filepath.Join(t.TempDir(), "secret.yaml")
	content := `apiVersion: v1
kind: Secret
metadata:
  name: force-secret
  namespace: prod
data:
  key: dmFsdWU=
`
	_ = os.WriteFile(yamlFile, []byte(content), 0644)

	// First import.
	if _, err := runCmd(t, "import", "--db", path, yamlFile); err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Second import without --force should fail.
	if _, err := runCmd(t, "import", "--db", path, yamlFile); err == nil {
		t.Fatal("expected error on duplicate import without --force")
	}

	// Second import with --force should succeed.
	if _, err := runCmd(t, "import", "--db", path, "--force", yamlFile); err != nil {
		t.Fatalf("force import via cmd: %v", err)
	}

	db, _ := kdbx.OpenDB(path, "pw")
	if got := len(db.ListEntries("")); got != 1 {
		t.Errorf("expected 1 entry after force import, got %d", got)
	}
}
