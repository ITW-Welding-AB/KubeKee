package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ITW-Welding-AB/KubeKee/internal/kdbx"
	"github.com/spf13/cobra"
)

var attrGroup string

var attrCmd = &cobra.Command{
	Use:   "attr",
	Short: "Manage additional attributes on KeePass entries",
}

// attr set <entry> <key>=<value> [<key>=<value> ...]
var attrSetCmd = &cobra.Command{
	Use:   "set <entry-title> <key>=<value> [<key>=<value> ...]",
	Short: "Set (upsert) one or more attributes on an entry",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}
		title := args[0]
		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}
		for _, pair := range args[1:] {
			k, v, ok := strings.Cut(pair, "=")
			if !ok || k == "" {
				return fmt.Errorf("invalid key=value pair: %q", pair)
			}
			if err := db.SetAttribute(title, attrGroup, k, v); err != nil {
				return err
			}
			fmt.Printf("Set attribute %q on entry %q\n", k, title)
		}
		return db.Save()
	},
}

// attr get <entry> [key ...]  — prints all attrs or the requested keys
var attrGetCmd = &cobra.Command{
	Use:   "get <entry-title> [key ...]",
	Short: "Get attributes of an entry (all, or specific keys)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}
		title := args[0]
		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}
		entry, err := db.GetEntry(title, attrGroup)
		if err != nil {
			return err
		}
		if len(args) > 1 {
			for _, k := range args[1:] {
				v, ok := entry.Attributes[k]
				if !ok {
					fmt.Fprintf(os.Stderr, "attribute %q not found\n", k)
					continue
				}
				fmt.Printf("%s=%s\n", k, v)
			}
			return nil
		}
		// Print all sorted
		keys := make([]string, 0, len(entry.Attributes))
		for k := range entry.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tVALUE")
		for _, k := range keys {
			fmt.Fprintf(w, "%s\t%s\n", k, entry.Attributes[k])
		}
		w.Flush()
		return nil
	},
}

// attr delete <entry> <key> [keys...]
var attrDeleteCmd = &cobra.Command{
	Use:     "delete <entry-title> <key> [key ...]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete one or more attributes from an entry",
	Args:    cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("--db is required")
		}
		title := args[0]
		db, err := kdbx.OpenDB(dbPath, getPassword())
		if err != nil {
			return err
		}
		for _, k := range args[1:] {
			if err := db.DeleteAttribute(title, attrGroup, k); err != nil {
				return err
			}
			fmt.Printf("Deleted attribute %q from entry %q\n", k, title)
		}
		return db.Save()
	},
}

func init() {
	attrCmd.PersistentFlags().StringVar(&attrGroup, "group", "", "KeePass group to search in")
	attrCmd.AddCommand(attrSetCmd)
	attrCmd.AddCommand(attrGetCmd)
	attrCmd.AddCommand(attrDeleteCmd)
	rootCmd.AddCommand(attrCmd)
}
