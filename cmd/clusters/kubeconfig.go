package clusters

import (
	"fmt"

	clusterspkg "github.com/openmcp-project/ocpctl/pkg/clusters"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	"github.com/spf13/cobra"
)

func newKubeconfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Manage kubeconfigs for clusters",
	}

	cmd.AddCommand(
		newKubeconfigExportCmd(),
		newKubeconfigGetCmd(),
	)

	return cmd
}

func newKubeconfigExportCmd() *cobra.Command {
	var environment string
	var name string
	var kubeconfig string
	var internal bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export kubeconfig to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			err := clusterspkg.Export(ctx, environment, name, kubeconfig, internal, kind.NewProviderKind())
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&environment, "environment", "e", "", "environment name")
	cmd.Flags().StringVarP(&name, "name", "n", "", "cluster name")
	cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "sets kubeconfig path instead of $KUBECONFIG or $HOME/.kube/config")
	cmd.Flags().BoolVarP(&internal, "internal", "i", false, "internal kubeconfig")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.RegisterFlagCompletionFunc("kubeconfig", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

func newKubeconfigGetCmd() *cobra.Command {
	var environment string
	var name string
	var internal bool

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Print kubeconfig to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			config, err := clusterspkg.Get(ctx, environment, name, internal, kind.NewProviderKind())
			if err != nil {
				return err
			}
			fmt.Print(config)
			return nil
		},
	}

	cmd.Flags().StringVarP(&environment, "environment", "e", "", "environment name")
	cmd.Flags().StringVarP(&name, "name", "n", "", "cluster name")
	cmd.Flags().BoolVarP(&internal, "internal", "i", false, "internal kubeconfig")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
