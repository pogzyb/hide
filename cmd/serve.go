package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pogzyb/hide/proxy"
)

var (
	commandServe = &cobra.Command{
		Use: "serve",
		Run: func(cmd *cobra.Command, args []string) {
			proxy.Run(port)
		},
	}

	port string
)

func init() {
	commandServe.Flags().StringVar(&port, "port", "8181", "The port which the proxy will listen on.")
	commandRoot.AddCommand(commandServe)
}

