package cli

import (
	"fmt"
	"os"

	"github.com/ITW-Welding-AB/KubeKee/api/v1alpha1"
	"github.com/ITW-Welding-AB/KubeKee/internal/operator"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Run the KubeKee Kubernetes operator",
	RunE: func(cmd *cobra.Command, args []string) error {
		scheme := runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(v1alpha1.AddToScheme(scheme))

		ctrl.SetLogger(zap.New())

		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme: scheme,
		})
		if err != nil {
			return fmt.Errorf("creating manager: %w", err)
		}

		if err := (&operator.KeePassSourceReconciler{
			Client: mgr.GetClient(),
		}).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("setting up controller: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Starting KubeKee operator...")
		return mgr.Start(ctrl.SetupSignalHandler())
	},
}

func init() {
	rootCmd.AddCommand(operatorCmd)
}
