package cli

import (
	"fmt"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new KeePass database",
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}
		if err := kdbx.CreateDB(dbPath, getPassword()); err != nil {
			return err
		}
		fmt.Printf("Created KeePass database: %s\n", dbPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
