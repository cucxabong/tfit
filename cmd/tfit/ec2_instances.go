package main

import (
	"github.com/spf13/cobra"
)

func NewCmdEC2Instances() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instances",
		Short: "EC2 Instances",
		Run: func(cmd *cobra.Command, args []string) {
			ec2, err := c.GetInstances()
			handleError(err)
			handleError(ec2.WriteHCL(w))
		},
	}

	return cmd
}
