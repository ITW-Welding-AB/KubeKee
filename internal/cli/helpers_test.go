package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
)

// newTestDB creates a fresh kdbx file in a temp dir and returns its path.
func newTestDB(t *testing.T, pass string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.kdbx")
	if err := kdbx.CreateDB(path, pass); err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	return path
}

// runCmd executes rootCmd with the given arguments, captures real os.Stdout
// output (needed for commands that write via tabwriter/fmt.Print directly),
// and returns the captured string along with any cobra error.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	rootCmd.SetOut(w)
	rootCmd.SetErr(w)
	rootCmd.SetArgs(args)

	_, execErr := rootCmd.ExecuteC()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("reading pipe: %v", readErr)
	}
	r.Close()

	return buf.String(), execErr
}
