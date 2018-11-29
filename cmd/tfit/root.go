package main

import (
	"fmt"
	"io"
	"os"

	"github.com/d0m0reg00dthing/tfit/pkg/tfit"
	"github.com/spf13/cobra"
)

type RootCmd struct {
	cobraCommand *cobra.Command
	cfg          tfit.Config
}

var c *tfit.AWSClient
var output string
var w io.Writer

var rootCommand = RootCmd{
	cobraCommand: &cobra.Command{
		Use: "tfit",
	},
}

func Execute() {
	if err := rootCommand.cobraCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	NewRootCmd()
}

func NewRootCmd() *cobra.Command {
	cmd := rootCommand.cobraCommand

	defaultAccesKey := os.Getenv("AWS_ACCESS_KEY_ID")
	cmd.PersistentFlags().StringVar(&rootCommand.cfg.AccessKey, "access-key", defaultAccesKey, "AWS Access Key ID. Overrides AWS_ACCESS_KEY_ID environment variable")
	defaultSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	cmd.PersistentFlags().StringVar(&rootCommand.cfg.SecretKey, "secret-key", defaultSecretKey, "AWS Secret Key. Overrides AWS_SECRET_ACCESS_KEY environment variable")

	defaultRegion := os.Getenv("AWS_REGION")
	cmd.PersistentFlags().StringVar(&rootCommand.cfg.Region, "region", defaultRegion, "AWS Region. Overrides AWS_REGION environment variable")

	defaultProfile := os.Getenv("AWS_PROFILE")
	cmd.PersistentFlags().StringVar(&rootCommand.cfg.Profile, "profile", defaultProfile, "AWS Profile. Overrides AWS_PROFILE environment variable")

	cmd.PersistentFlags().StringVar(&output, "output", "", "The output of HCL (Terraform config) contents (Default to StdOut)")

	// Sub-commands
	cmd.AddCommand(NewCmdEC2())
	cmd.AddCommand(NewCmdRoute53())
	cmd.AddCommand(NewCmdIAM())
	cmd.AddCommand(NewCmdS3())
	cmd.AddCommand(NewCmdAutoScaling())
	cmd.AddCommand(NewCmdELB())

	return cmd
}

func initConfig() {
	var err error
	c, err = rootCommand.cfg.Client()
	handleError(err)

	if len(output) == 0 {
		w = os.Stdout
	} else {
		w, err = os.OpenFile(output, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		handleError(err)
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
