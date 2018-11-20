package main

import (
	"github.com/spf13/cobra"
)

func NewCmdRoute53Zones() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zone",
		Short: "Route53 Hosted Zones",
		Run: func(cmd *cobra.Command, args []string) {
			zones, err := c.GetHostZones(5)
			handleError(err)
			handleError(zones.WriteHCL(w))
		},
	}

	return cmd
}
