package takeover

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/hexinatgithub/takeover/pkg/client"
	"github.com/hexinatgithub/takeover/pkg/cmd/server"
)

func NewCommand(name string) *cobra.Command {
	c := &cobra.Command{
		Use:   name,
		Short: "Takeover product Kubernetes cluster after crash, restore resources and data, bring cluster state back to product.",
	}

	config, err := client.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error reading config file: %v\n", err)
	}

	f := client.NewFactory(name, config)
	f.BindFlags(c.PersistentFlags())

	c.AddCommand(
		server.NewCommand(f),
	)

	klog.InitFlags(flag.CommandLine)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
