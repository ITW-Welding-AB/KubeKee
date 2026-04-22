package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath           string
	password         string
	annotationPrefix = "kubekee."
)

var rootCmd = &cobra.Command{
	Use:   "kubekee",
	Short: "KubeKee - K8s KeePass CLI & Operator for CI/CD workflows",
	Long: `KubeKee manages Kubernetes manifests (secrets, configmaps, etc.)
inside KeePass databases for secure, version-controlled storage.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "path to KeePass database file")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "database password (prefer KUBEKEE_PASSWORD env var)")
}

func getPassword() string {
	if password != "" {
		return password
	}
	if p := os.Getenv("KUBEKEE_PASSWORD"); p != "" {
		return p
	}
	fmt.Fprintln(os.Stderr, "Error: password required via --password or KUBEKEE_PASSWORD env var")
	os.Exit(1)
	return ""
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
