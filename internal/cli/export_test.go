package cli

import (
	"strings"
	"testing"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
)

func TestInjectAnnotations_NoAttributes(t *testing.T) {
	entry := &kdbx.Entry{
		Content:    "apiVersion: v1\nkind: Secret\n",
		Attributes: map[string]string{},
	}
	out, err := injectAnnotations(entry)
	if err != nil {
		t.Fatal(err)
	}
	if out != entry.Content {
		t.Errorf("expected unchanged content, got %q", out)
	}
}

func TestInjectAnnotations_AddsAnnotations(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: val
`
	entry := &kdbx.Entry{
		Content:    content,
		Attributes: map[string]string{"version": "v1.0.0"},
	}
	out, err := injectAnnotations(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "kubekee.version") {
		t.Errorf("expected kubekee.version annotation in output:\n%s", out)
	}
	if !strings.Contains(out, "v1.0.0") {
		t.Errorf("expected v1.0.0 in output:\n%s", out)
	}
}

func TestInjectAnnotations_UpdatesExisting(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    kubekee.version: old
data: {}
`
	entry := &kdbx.Entry{
		Content:    content,
		Attributes: map[string]string{"version": "new"},
	}
	out, err := injectAnnotations(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "old") {
		t.Errorf("expected old value to be replaced:\n%s", out)
	}
	if !strings.Contains(out, "new") {
		t.Errorf("expected new value in output:\n%s", out)
	}
}

func TestInjectAnnotations_NotYAML(t *testing.T) {
	entry := &kdbx.Entry{
		Content:    "not valid yaml: [[[",
		Attributes: map[string]string{"k": "v"},
	}
	out, err := injectAnnotations(entry)
	if err != nil {
		t.Fatal(err)
	}
	if out != entry.Content {
		t.Errorf("expected unchanged content for invalid YAML")
	}
}
