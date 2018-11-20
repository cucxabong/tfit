package main

import (
	"github.com/spf13/cobra"
)

func NewCmdRoute53() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route53",
		Short: "Route53 Hosted Zones & Resource Record Sets",
	}

	cmd.AddCommand(NewCmdRoute53Zones())
	cmd.AddCommand(NewCmdRoute53ResourceRecordSet())

	return cmd
}
