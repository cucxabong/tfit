package main

import (
	"github.com/spf13/cobra"
)

func NewCmdS3Buckets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buckets",
		Short: "S3 Buckets",
		Run: func(cmd *cobra.Command, args []string) {
			buckets, err := c.GetBuckets()
			handleError(err)
			handleError(buckets.WriteHCL(w))
		},
	}

	return cmd
}
