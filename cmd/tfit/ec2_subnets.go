package main

import (
	"github.com/spf13/cobra"
)

func NewCmdEC2Subnets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subnet",
		Short: "EC2 Subnet",
		Run: func(cmd *cobra.Command, args []string) {
			subnets, err := c.GetSubnets()
			handleError(err)
			handleError(subnets.WriteHCL(w))
		},
	}

	return cmd
}
