package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	commandRoot = &cobra.Command{
		Use: "hide",
		Short: "Hide Proxy CLI.",
		Long: "Manage your Hide Proxy with this tool.",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("...")
		},
	}
)

func Execute() {
	if err := commandRoot.Execute(); err != nil {
		log.Fatal().Msgf("could not execute command: %v", err)
	}
}