package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is optionally injected at build time:
//
//	-ldflags "-X github.com/ITW-Welding-AB/KubeKee/internal/cli.version=v1.2.3"
//
// When installed via `go install module@vX.Y.Z`, the Go toolchain embeds the
// module version in the binary automatically and the fallback below picks it up.
var version = ""

// Version returns the effective version string, preferring the injected ldflags
// value, then the module build-info version (set by `go install`), then "dev".
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the KubeKee version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
