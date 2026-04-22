package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var importGroup string
var importForce bool

var importCmd = &cobra.Command{
	Use:   "import <file.yaml|file.json> [files...]",
	Short: "Import YAML/JSON files as entries in the KeePass database",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}

		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}

		for _, file := range args {
			if err := importFile(db, file, importForce); err != nil {
				return fmt.Errorf("importing %s: %w", file, err)
			}
		}

		if err := db.Save(); err != nil {
			return err
		}

		fmt.Printf("Imported %d file(s) into %s\n", len(args), dbPath)
		return nil
	},
}

func importFile(db *kdbx.DB, filePath string, force bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Parse to extract metadata (including annotations)
	var meta struct {
		Kind     string `yaml:"kind" json:"kind"`
		Metadata struct {
			Name        string            `yaml:"name" json:"name"`
			Namespace   string            `yaml:"namespace" json:"namespace"`
			Annotations map[string]string `yaml:"annotations" json:"annotations"`
		} `yaml:"metadata" json:"metadata"`
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &meta); err != nil {
			// Store as raw content even if not valid K8s manifest
			meta.Kind = "Raw"
			meta.Metadata.Name = filepath.Base(filePath)
		}
	case ".json":
		if err := json.Unmarshal(data, &meta); err != nil {
			meta.Kind = "Raw"
			meta.Metadata.Name = filepath.Base(filePath)
		}
	default:
		meta.Kind = "Raw"
		meta.Metadata.Name = filepath.Base(filePath)
	}

	title := meta.Metadata.Name
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	group := importGroup
	if group == "" && meta.Metadata.Namespace != "" {
		group = meta.Metadata.Namespace
	}

	// Seed attributes with kubekee lifecycle stamps.
	attrs := map[string]string{
		"version":    Version(),
		"createdAt":  time.Now().UTC().Format(time.RFC3339),
		"modifiedAt": time.Now().UTC().Format(time.RFC3339),
	}

	// Promote annotations from the manifest into entry attributes.
	// Annotations that already carry the kubekee. prefix are stored without the prefix
	// (they are kubekee-owned). All other annotations are stored verbatim.
	for k, v := range meta.Metadata.Annotations {
		attrKey := k
		if strings.HasPrefix(k, annotationPrefix) {
			attrKey = strings.TrimPrefix(k, annotationPrefix)
		}
		// Don't overwrite lifecycle stamps already set above.
		if _, exists := attrs[attrKey]; !exists {
			attrs[attrKey] = v
		}
	}

	entry := kdbx.Entry{
		Title:      title,
		Group:      group,
		Content:    string(data),
		Kind:       meta.Kind,
		Name:       meta.Metadata.Name,
		Namespace:  meta.Metadata.Namespace,
		Attributes: attrs,
	}

	if force {
		err := db.UpdateEntry(title, group, string(data), attrs)
		if err == nil {
			return nil
		}
		// Entry doesn't exist yet; fall through to add it.
	}

	return db.AddEntry(entry)
}

func init() {
	importCmd.Flags().StringVar(&importGroup, "group", "", "KeePass group/namespace to store entries in")
	importCmd.Flags().BoolVar(&importForce, "force", false, "Overwrite an existing entry instead of returning an error")
	rootCmd.AddCommand(importCmd)
}
