package cmd

import (
	"context"

	"github.com/pogzyb/hide/infra/aws"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	commandDeploy = &cobra.Command{
		Use: "deploy",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("hi")
		},
	}

	commandDeployAWS = &cobra.Command{
		Use: "aws",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			deployAWS(ctx)
		},
	}

	ipAddr string

	awsVpcId    string
	awsSubnetId string
)

func init() {
	commandDeploy.PersistentFlags().StringVar(&ipAddr, "ipAddr", "", "Your IP address. Incoming traffic to hide-proxy will be limited to this IP. If not specified, Hide will query whatsmyip.com.")
	commandDeploy.MarkPersistentFlagRequired("ipAddr")
	commandRoot.AddCommand(commandDeploy)

	commandDeployAWS.Flags().StringVar(&awsVpcId, "vpcId", "", "The Id of VPC where the Hide Proxy EC2 will be launched.")
	commandDeployAWS.Flags().StringVar(&awsSubnetId, "subnetId", "", "The Id of Public Subnet where the Hide Proxy EC2 will be launched.")
	commandDeploy.AddCommand(commandDeployAWS)
}

func deployAWS(ctx context.Context) {
	provider, err := aws.NewProvider(ctx, ipAddr, awsVpcId, awsSubnetId)
	if err != nil {
		log.Fatal().Msgf("could not get provider: %v", err)
	}
	info, err := provider.Deploy(ctx)
	if err != nil {
		log.Fatal().Msgf("could not deploy: %v", err)
	}
	log.Info().Msgf("hostname: %s", info.Hostname)
}
