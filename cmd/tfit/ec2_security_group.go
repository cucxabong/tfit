package main

import (
	"github.com/spf13/cobra"
)

func NewCmdEC2SecurityGroups() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secgroup",
		Short: "EC2 Security Groups",
		Run: func(cmd *cobra.Command, args []string) {
			AccountId, err := rootCommand.cfg.GetAccountId()
			handleError(err)
			sg, err := c.GetSecurityGroups(AccountId)
			handleError(err)
			handleError(sg.WriteHCL(w))
		},
	}

	return cmd
}
