package main

import (
	"github.com/spf13/cobra"
)

func NewCmdEC2VPCs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "EC2 VPC",
		Run: func(cmd *cobra.Command, args []string) {
			vpc, err := c.GetVPCs()
			handleError(err)
			handleError(vpc.WriteHCL(w))
		},
	}

	return cmd
}
