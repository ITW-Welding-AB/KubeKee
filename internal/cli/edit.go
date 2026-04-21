package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
)

var editGroup string

var editCmd = &cobra.Command{
	Use:   "edit <entry-title>",
	Short: "Edit an existing KeePass entry using $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}

		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}

		entry, err := db.GetEntry(args[0], editGroup)
		if err != nil {
			return err
		}

		// Write to temp file
		tmp, err := os.CreateTemp("", "kubekee-*.yaml")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		defer os.Remove(tmp.Name())

		if _, err := tmp.WriteString(entry.Content); err != nil {
			tmp.Close()
			return err
		}
		tmp.Close()

		// Open editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editorCmd := exec.Command(editor, tmp.Name())
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		// Read back
		newContent, err := os.ReadFile(tmp.Name())
		if err != nil {
			return err
		}

		if string(newContent) == entry.Content {
			fmt.Println("No changes made.")
			return nil
		}

		if err := db.UpdateEntry(args[0], editGroup, string(newContent), map[string]string{
			"kubekee.version":    Version(),
			"kubekee.modifiedAt": time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}

		if err := db.Save(); err != nil {
			return err
		}

		fmt.Printf("Updated entry %q\n", args[0])
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editGroup, "group", "", "KeePass group to search in")
	rootCmd.AddCommand(editCmd)
}
