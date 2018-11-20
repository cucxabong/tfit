package tfit

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Instance is a shorter version if ec2.Instance
type Instance struct {
	EbsOptimized       *bool
	IamInstanceProfile *string
	ImageID            *string
	InstanceID         *string
	InstanceType       *string
	KeyName            *string
	Monitoring         *bool
	SecurityGroups     []*string
	SourceDestCheck    *bool
	SubnetID           *string
	VpcID              *string
	Tags               map[*string]*string
}

// A group of Instance
type Instances []*Instance

func (i *Instance) set(src *ec2.Instance) error {
	i.EbsOptimized = src.EbsOptimized

	if src.IamInstanceProfile != nil && src.IamInstanceProfile.Arn != nil {
		tmp := strings.Split(aws.StringValue(src.IamInstanceProfile.Arn), "/")
		i.IamInstanceProfile = aws.String(tmp[len(tmp)-1])
	}

	i.ImageID = src.ImageId
	i.InstanceID = src.InstanceId
	i.InstanceType = src.InstanceType
	i.KeyName = src.KeyName
	if strings.Compare(aws.StringValue(src.Monitoring.State), "disabled") == 0 {
		i.Monitoring = aws.Bool(false)
	} else {
		i.Monitoring = aws.Bool(true)
	}

	// Build []*string from []*ec2.GroupIdentifier
	if src.SecurityGroups != nil {
		for _, sg := range src.SecurityGroups {
			i.SecurityGroups = append(i.SecurityGroups, sg.GroupName)
		}
	}

	i.SourceDestCheck = src.SourceDestCheck
	i.SubnetID = src.SubnetId
	i.VpcID = src.VpcId

	// Build map[*]*string from []*ec2.Tag
	if src.Tags != nil {
		i.Tags = make(map[*string]*string)
		for _, t := range src.Tags {
			i.Tags[t.Key] = t.Value
		}
	}

	return nil
}

func (i *Instances) set(src []*ec2.Instance) {
	if src == nil {
		return
	}

	for _, v := range src {
		// Check if instance's state is 'terminated'
		// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#InstanceState
		if aws.Int64Value(v.State.Code) == 48 {
			continue
		}

		tmp := &Instance{}
		tmp.set(v)
		*i = append(*i, tmp)
	}
}

// DescribeAllInstances ...
func (c *AWSClient) GetInstances() (*Instances, error) {
	ec2conn := c.ec2conn
	instances := &Instances{}

	opt := &ec2.DescribeInstancesInput{}
	for {
		out, err := ec2conn.DescribeInstances(opt)
		if err != nil {
			return nil, err
		}

		for _, rsv := range out.Reservations {
			instances.set(rsv.Instances)
		}

		if out.NextToken != nil {
			opt.NextToken = out.NextToken
			fmt.Println(aws.StringValue(opt.NextToken))
		} else {
			//fmt.Println("Breaking.......")
			break
		}
	}

	return instances, nil
}

// Render will render terraform format from 'Instances'
func (i *Instances) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"StringValueSlice": aws.StringValueSlice,
	}

	tmpl := `
	{{ if . }}
		{{ range . }}
	resource "aws_instance" "{{ .InstanceID }}_instance" {
		ami = "{{ .ImageID }}"
		instance_type = "{{ .InstanceType }}"
		{{- if .EbsOptimized }}
		ebs_optimized = {{ .EbsOptimized }}
		{{- end }}
		{{- if .IamInstanceProfile }}
		iam_instance_profile = "{{ .IamInstanceProfile }}"
		{{- end }}
		{{- if .KeyName}}
		key_name = "{{ .KeyName }}"
		{{- end }}
		{{- if .Monitoring }}
		monitoring = {{.Monitoring}}
		{{- end}}
		{{- if .SourceDestCheck }}
		source_dest_check = {{ .SourceDestCheck }}
    {{- end}}
    {{- if .SubnetID}}
    subnet_id = "{{ .SubnetID }}"
    {{- end}}
    {{- if .SecurityGroups }}
    {{- $secgroup := StringValueSlice .SecurityGroups }}
    vpc_security_group_ids = [{{ $secgroup | joinstring "," }}]
    {{- end}}
    {{if .Tags}}
    tags {
      {{range $k, $v := .Tags}}
        "{{ $k }}" = "{{$v}}"
      {{- end}}
    }
    {{end}}
	}
		{{- end}}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, i)

}

//**************** VPC ****************
// https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-vpc-attribute.html
// func (c *EC2) DescribeVpcAttribute(input *DescribeVpcAttributeInput) (*DescribeVpcAttributeOutput, error)

//**************** Keypair ****************

// https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-security-groups.html
// https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-subnets.html
