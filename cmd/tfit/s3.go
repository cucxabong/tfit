package main

import (
	"github.com/spf13/cobra"
)

func NewCmdS3() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3",
		Short: "S3 Related resources",
	}

	cmd.AddCommand(NewCmdS3Buckets())

	return cmd
}
