package cli

import (
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "admin-console",
		Short:         "Make the admin console accessible",
		Long:          "Establish port forwarding for localhost access to the kotsadm admin console.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewLogger()

			podName, err := k8sutil.WaitForWeb(v.GetString("namespace"), time.Second*5)
			if err != nil {
				return err
			}

			stopCh, err := k8sutil.PortForward(v.GetString("kubeconfig"), 8800, 3000, v.GetString("namespace"), podName, true)
			if err != nil {
				return err
			}
			defer close(stopCh)

			log.ActionWithoutSpinner("Press Ctrl+C to exit")
			log.ActionWithoutSpinner("Go to http://localhost:8800 to access the Admin Console")

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)

			<-signalChan

			log.ActionWithoutSpinner("Cleaning up")

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")
	cmd.Flags().StringP("namespace", "n", "default", "the namespace where the admin console is running")

	cmd.AddCommand(AdminConsoleUpgradeCmd())

	return cmd
}
