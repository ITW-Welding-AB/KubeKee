package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
)

var listGroup string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all entries in the KeePass database",
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}

		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}

		entries := db.ListEntries(listGroup)
		if len(entries) == 0 {
			fmt.Println("No entries found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "GROUP\tTITLE\tKIND")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Group, e.Title, e.Kind)
		}
		w.Flush()
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listGroup, "group", "", "filter by group")
	rootCmd.AddCommand(listCmd)
}
