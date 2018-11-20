package main

import (
	"github.com/spf13/cobra"
)

func NewCmdRoute53ResourceRecordSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rrs",
		Short: "Route53 Resource Record Sets",
		Run: func(cmd *cobra.Command, args []string) {
			rrs, err := c.GetAllResourceRecordSets()
			handleError(err)
			handleError(rrs.WriteHCL(w))
		},
	}

	return cmd
}
