package main

import (
	"github.com/spf13/cobra"
)

func NewCmdAutoScaling() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "as",
		Short: "AutoScaling Related",
	}

	cmd.AddCommand(NewCmdAutoScalingLaunchConfiguration())
	cmd.AddCommand(NewCmdAutoScalingAutoScalingGroup())

	return cmd
}
