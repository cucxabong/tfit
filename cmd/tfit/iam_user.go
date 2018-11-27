package main

import (
	"github.com/spf13/cobra"
)

func NewCmdIAMUser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "IAM Users",
		Run: func(cmd *cobra.Command, args []string) {
			users, err := c.ListUsers()
			handleError(err)
			handleError(users.WriteHCL(w))
		},
	}

	return cmd
}
