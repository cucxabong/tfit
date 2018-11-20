package main

import (
	"github.com/spf13/cobra"
)

func NewCmdAutoScalingLaunchConfiguration() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lc",
		Short: "Launch Configuration",
		Run: func(cmd *cobra.Command, args []string) {
			launchConfigs, err := c.GetLaunchConfigurations()

			handleError(err)
			handleError(launchConfigs.WriteHCL(w))
		},
	}

	return cmd
}
