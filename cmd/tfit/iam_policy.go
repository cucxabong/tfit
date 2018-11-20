package main

import (
	"github.com/spf13/cobra"
)

func NewCmdIAMPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "IAM Policies",
		Run: func(cmd *cobra.Command, args []string) {
			polices, err := c.GetPolicies()
			handleError(err)
			handleError(polices.WriteHCL(w))
		},
	}

	return cmd
}
