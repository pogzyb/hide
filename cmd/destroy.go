package cmd

import (
	"context"

	"github.com/pogzyb/hide/infra/aws"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	commandDestroy = &cobra.Command{
		Use: "destroy",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("hi")
		},
	}

	commandDestroyAWS = &cobra.Command{
		Use: "aws",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			destroyAWS(ctx)
		},
	}
)

func init() {
	commandRoot.AddCommand(commandDestroy)
	commandDestroy.AddCommand(commandDestroyAWS)
}

func destroyAWS(ctx context.Context) {
	provider, err := aws.NewProvider(ctx, "", "", "")
	if err != nil {
		log.Fatal().Msgf("could not get provider: %v", err)
	}
	err = provider.Destroy(ctx)
	if err != nil {
		log.Fatal().Msgf("could not destroy: %v", err)
	}
	log.Info().Msgf("destroyed")
}
