package main

import (
	"github.com/spf13/cobra"
)

func NewCmdEC2() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ec2",
		Short: "EC2 Related",
	}

	cmd.AddCommand(NewCmdEC2Instances())
	cmd.AddCommand(NewCmdEC2SecurityGroups())
	cmd.AddCommand(NewCmdEC2VPCs())
	cmd.AddCommand(NewCmdEC2Subnets())

	return cmd
}
