package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	exportGroup  string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export <entry-title>",
	Short: "Export a KeePass entry back to YAML/JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}

		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}

		entry, err := db.GetEntry(args[0], exportGroup)
		if err != nil {
			return err
		}

		content, err := injectAnnotations(entry)
		if err != nil {
			return fmt.Errorf("injecting annotations: %w", err)
		}

		if exportOutput == "" || exportOutput == "-" {
			fmt.Print(content)
			return nil
		}

		if err := os.WriteFile(exportOutput, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		fmt.Printf("Exported %q to %s\n", args[0], exportOutput)
		return nil
	},
}

// injectAnnotations takes an entry's attributes and writes them back into
// metadata.annotations of the stored YAML/JSON content.
// kubekee-owned attributes (e.g. "version", "createdAt") are written as
// "kubekee.<key>". All other attributes are written verbatim.
// If the content is not a valid YAML/JSON Kubernetes manifest the original
// content is returned unchanged.
func injectAnnotations(entry *kdbx.Entry) (string, error) {
	if len(entry.Attributes) == 0 {
		return entry.Content, nil
	}

	// Attempt YAML parse (covers both YAML and JSON since JSON is valid YAML).
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(entry.Content), &doc); err != nil || doc.Kind == 0 {
		// Not parseable — return as-is.
		return entry.Content, nil
	}

	// doc is a Document node; the actual mapping is its first child.
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return entry.Content, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return entry.Content, nil
	}

	// Build the annotations map to inject: kubekee-owned attrs get the prefix.
	annotations := map[string]string{}
	for k, v := range entry.Attributes {
		annotationKey := k
		// Keys that don't already contain a "/" (not a domain-qualified annotation)
		// and don't already start with the prefix get the kubekee. prefix.
		if !strings.Contains(k, "/") && !strings.HasPrefix(k, annotationPrefix) {
			annotationKey = annotationPrefix + k
		}
		annotations[annotationKey] = v
	}

	// Walk the YAML AST to find/create metadata.annotations.
	setAnnotationsInNode(root, annotations)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return entry.Content, nil
	}
	return string(out), nil
}

// setAnnotationsInNode mutates a YAML mapping node to set metadata.annotations.
func setAnnotationsInNode(root *yaml.Node, annotations map[string]string) {
	// Find the "metadata" key.
	var metaValue *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "metadata" {
			metaValue = root.Content[i+1]
			break
		}
	}
	if metaValue == nil {
		// No metadata node — nothing to inject into.
		return
	}
	if metaValue.Kind != yaml.MappingNode {
		return
	}

	// Find or create the "annotations" key inside metadata.
	var annotationsValue *yaml.Node
	for i := 0; i+1 < len(metaValue.Content); i += 2 {
		if metaValue.Content[i].Value == "annotations" {
			annotationsValue = metaValue.Content[i+1]
			break
		}
	}
	if annotationsValue == nil {
		// Create the annotations mapping.
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "annotations"}
		valNode := &yaml.Node{Kind: yaml.MappingNode}
		metaValue.Content = append(metaValue.Content, keyNode, valNode)
		annotationsValue = valNode
	}

	// Upsert each annotation into the mapping node.
	for k, v := range annotations {
		updated := false
		for i := 0; i+1 < len(annotationsValue.Content); i += 2 {
			if annotationsValue.Content[i].Value == k {
				annotationsValue.Content[i+1].Value = v
				updated = true
				break
			}
		}
		if !updated {
			annotationsValue.Content = append(annotationsValue.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				&yaml.Node{Kind: yaml.ScalarNode, Value: v},
			)
		}
	}
}

func init() {
	exportCmd.Flags().StringVar(&exportGroup, "group", "", "KeePass group to search in")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (default: stdout)")
	rootCmd.AddCommand(exportCmd)
}
