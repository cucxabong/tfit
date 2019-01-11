package main

import (
	"github.com/spf13/cobra"
)

func NewCmdEC2RouteTables() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rtb",
		Short: "VPC Route & Route Table",
		Run: func(cmd *cobra.Command, args []string) {
			rtb, err := c.GetRouteTables()
			handleError(err)
			handleError(rtb.WriteHCL(w))
		},
	}

	return cmd
}
