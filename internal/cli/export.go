package cli

import (
	"fmt"
	"os"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
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

		if exportOutput == "" || exportOutput == "-" {
			fmt.Print(entry.Content)
			return nil
		}

		if err := os.WriteFile(exportOutput, []byte(entry.Content), 0644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		fmt.Printf("Exported %q to %s\n", args[0], exportOutput)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportGroup, "group", "", "KeePass group to search in")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (default: stdout)")
	rootCmd.AddCommand(exportCmd)
}
