Export existing AWS resources to [Terraform](https://terraform.io/) style (HCL)

- [What's this](#whats-this)
- [Supported Resources](#supported-resources)
- [Installation](#installation)
- [Usage](#usage)
  -  [CLI](#cli)
  -  [Library](#library)

## What's this
Inspired by [terraforming](https://terraforming.dtan4.net) & for learning purpose, I re-write that tool in Go (in form of library & CLI). Any feedbacks & suggestions are welcomed.

## Supported Resources
* EC2
  * Instances
* Auto Scaling
  * Auto Scaling Group
  * Launch Configuration
* Route53
  * Hosted Zone
  * Resource Record Set
* IAM
  * Policy
* S3
  * Bucket
* **Updating ......**

## Installation
```bash
$ go get github.com/d0m0reg00dthing/tfit/cmd/tfit
```
## Usage
### CLI
```bash
$ tfit
Usage:
  tfit [command]

Available Commands:
  as          AutoScaling Related
  ec2         EC2 Related
  help        Help about any command
  iam         IAM Related
  route53     Route53 Hosted Zones & Resource Record Sets
  s3          S3 Related resources

Flags:
      --access-key string   AWS Access Key ID. Overrides AWS_ACCESS_KEY_ID environment variable
  -h, --help                help for tfit
      --output string       The output of HCL (Terraform config) contents (Default to StdOut)
      --profile string      AWS Profile. Overrides AWS_PROFILE environment variable
      --region string       AWS Region. Overrides AWS_REGION environment variable
      --secret-key string   AWS Secret Key. Overrides AWS_SECRET_ACCESS_KEY environment variable

Use "tfit [command] --help" for more information about a command.
```

#### Export S3 Buckets (Output to StdOut)
```bash
$ tfit --region us-east-1 --profile dev s3 buckets
```

```hcl
resource "aws_s3_bucket" "bar" {
  bucket = "bar"

  versioning {
    mfa_delete = false
  }
}

resource "aws_s3_bucket" "foo" {
  bucket = "foo"

  policy = <<POLICY
      {
 "Statement": [
  {
   "Action": "s3:GetObject",
   "Effect": "Allow",
   "Principal": {
    "AWS": "arn:aws:iam::cloudfront:user/foo"
   },
   "Resource": "arn:aws:s3:::foo-bucket/*",
   "Sid": "2"
  }
 ],
 "Version": "2008-10-17"
}
POLICY

  versioning {
    mfa_delete = false
  }

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET"]
    allowed_origins = ["*"]
    max_age_seconds = 3000
  }
}
```

#### Export EC2 Instances & write HCL to external file
```bash
tfit --region us-east-1 --profile dev --output instances.tf ec2 instances
```

### Library
```go
package main

import (
	"fmt"
	"os"

	"github.com/d0m0reg00dthing/tfit/pkg/tfit"
)

func main() {
	cfg := tfit.Config{
		Region:  "us-east-1",
		Profile: "dev",
	}

	c, err := cfg.Client()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get Route53 Hosted Zones
	zones, err := c.GetHostZones(5)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Pretty HCL format & write to StdOut
	if err = zones.WriteHCL(os.Stdout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```
