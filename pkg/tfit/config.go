package tfit

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Config struct {
	AccessKey string
	SecretKey string
	CredsFile string
	Profile   string
	Token     string
	Region    string
}

type AWSClient struct {
	r53conn *route53.Route53
	ec2conn *ec2.EC2
	iamconn *iam.IAM
	asconn  *autoscaling.AutoScaling
	s3conn  *s3.S3
}

func (c *Config) Client() (*AWSClient, error) {
	var client AWSClient
	creds := GetCredentials(c)

	sess, err := session.NewSession(&aws.Config{Credentials: creds})
	if err != nil {
		return nil, fmt.Errorf("Error creating AWS session: %s", err)
	}

	client.r53conn = route53.New(sess)
	client.iamconn = iam.New(sess)
	client.s3conn = s3.New(sess, aws.NewConfig().WithRegion(c.Region))

	client.ec2conn = ec2.New(sess, aws.NewConfig().WithRegion(c.Region))
	client.asconn = autoscaling.New(sess, aws.NewConfig().WithRegion(c.Region))

	return &client, nil
}
