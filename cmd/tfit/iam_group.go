package main

import (
	"github.com/spf13/cobra"
)

func NewCmdIAMGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "IAM Groups",
		Run: func(cmd *cobra.Command, args []string) {
			groups, err := c.ListIAMGroups()
			handleError(err)
			handleError(groups.WriteHCL(w))
		},
	}

	return cmd
}
