package main

import (
	"github.com/spf13/cobra"
)

func NewCmdELB() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "elb",
		Short: "Elastic Load Balancer",
		Run: func(cmd *cobra.Command, args []string) {
			elbs, err := c.ListELBs()
			handleError(err)
			handleError(elbs.WriteHCL(w))
		},
	}

	return cmd
}
