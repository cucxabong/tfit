package main

import (
	"github.com/spf13/cobra"
)

func NewCmdIAMRole() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "IAM Roles",
		Run: func(cmd *cobra.Command, args []string) {
			roles, err := c.ListRoles()
			handleError(err)
			handleError(roles.WriteHCL(w))
		},
	}

	return cmd
}
