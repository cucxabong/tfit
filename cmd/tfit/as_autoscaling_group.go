package main

import (
	"github.com/spf13/cobra"
)

func NewCmdAutoScalingAutoScalingGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asg",
		Short: "Auto Scaling Group",
		Run: func(cmd *cobra.Command, args []string) {
			groups, err := c.GetAutoScalingGroups()

			handleError(err)
			handleError(groups.WriteHCL(w))

		},
	}

	return cmd
}
