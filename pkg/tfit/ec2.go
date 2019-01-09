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
type VPC struct {
	// describe-vpcs
	CIDRBlock                    *string
	InstanceTenancy              *string
	Tags                         *Tags
	VPCId                        *string
	AssignGeneratedIPv6CIDRBlock *bool

	// describe-vpc-attribute
	EnableDnsHostnames *bool
	EnableDnsSupport   *bool

	// describe-vpc-classic-link
	EnableClassicLink *bool

	//describe-vpc-classic-link-dns-support
	EnableClassicLinkDnsSupport *bool
}

type VPCs []*VPC

func (c *AWSClient) setVPCAttribute(vpc *VPC, classicLink *ec2.DescribeVpcClassicLinkOutput, classicLinkDnsSupport *ec2.DescribeVpcClassicLinkDnsSupportOutput) error {
	opt := &ec2.DescribeVpcAttributeInput{
		VpcId: vpc.VPCId,
	}

	// EnableDnsHostnames
	opt = opt.SetAttribute("enableDnsHostnames")
	output, err := c.ec2conn.DescribeVpcAttribute(opt)
	if err != nil {
		return err
	}
	vpc.EnableDnsHostnames = output.EnableDnsHostnames.Value

	// EnableDnsSupport
	opt = opt.SetAttribute("enableDnsSupport")
	output, err = c.ec2conn.DescribeVpcAttribute(opt)
	if err != nil {
		return err
	}
	vpc.EnableDnsSupport = output.EnableDnsSupport.Value

	// EnableClassicLink
	for k, _ := range classicLink.Vpcs {
		if aws.StringValue(classicLink.Vpcs[k].VpcId) == aws.StringValue(vpc.VPCId) {
			vpc.EnableClassicLink = classicLink.Vpcs[k].ClassicLinkEnabled
			break
		}
	}

	//EnableClassicLinkDnsSupport
	for _, v := range classicLinkDnsSupport.Vpcs {
		if aws.StringValue(v.VpcId) == aws.StringValue(vpc.VPCId) {
			vpc.EnableClassicLinkDnsSupport = v.ClassicLinkDnsSupported
			break
		}
	}

	return nil
}

func (c *AWSClient) GetVPCs() (*VPCs, error) {
	res := VPCs{}

	basicInfo, err := c.ec2conn.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}

	classicLink, err := c.ec2conn.DescribeVpcClassicLink(&ec2.DescribeVpcClassicLinkInput{})
	if err != nil {
		return nil, err
	}

	classicLinkDnsSupport, err := c.ec2conn.DescribeVpcClassicLinkDnsSupport(&ec2.DescribeVpcClassicLinkDnsSupportInput{})
	if err != nil {
		return nil, err
	}

	for _, v := range basicInfo.Vpcs {
		vpc := VPC{
			CIDRBlock:       v.CidrBlock,
			InstanceTenancy: v.InstanceTenancy,
			VPCId:           v.VpcId,
			Tags:            &Tags{},
		}

		// Set Tags
		vpc.Tags.setTags(v.Tags)
		if len(v.Ipv6CidrBlockAssociationSet) > 0 {
			vpc.AssignGeneratedIPv6CIDRBlock = aws.Bool(true)
		}
		err = c.setVPCAttribute(&vpc, classicLink, classicLinkDnsSupport)
		if err != nil {
			return nil, err
		}

		res = append(res, &vpc)
	}

	return &res, nil
}

func (vpcs *VPCs) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{}

	tmpl := `
	{{ if . }}
		{{- range . }}
	resource "aws_vpc" "{{ index .Tags "Name" }}" {
    cidr_block = "{{ .CIDRBlock }}"
    {{- if .InstanceTenancy }}
    instance_tenancy = "{{ .InstanceTenancy}}"
    {{- end}}
    {{- if .Tags }}
    tags {
      {{range $k, $v := .Tags}}
        "{{ $k }}" = "{{$v }}"
      {{- end}}
    }
    {{- end }}
    {{- if .EnableDnsHostnames }}
    enable_dns_hostnames = {{ .EnableDnsHostnames}}
    {{- end }}
    {{- if .EnableDnsSupport}}
    enable_dns_support = {{.EnableDnsSupport}}
    {{- end}}
    {{- if .EnableClassicLink}}
    enable_classiclink = {{ .EnableClassicLink}}
    {{- end}}
    {{- if .EnableClassicLinkDnsSupport }}
    enable_classiclink_dns_support = {{ .EnableClassicLinkDnsSupport }}
    {{- end}}
    {{- if .AssignGeneratedIPv6CIDRBlock }}
    assign_generated_ipv6_cidr_block  = {{ .AssignGeneratedIPv6CIDRBlock}}
    {{- end}}
	}
		{{- end}}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, vpcs)

}

//**************** Subnet ****************
// https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-subnets.html
type Subnet struct {
	VPCId *string

	Tags                        *Tags
	MapPublicIpOnLaunch         *bool
	CIDRBlock                   *string
	IPv6CIDRBlock               *string
	AvailabilityZone            *string
	SubnetId                    *string
	AssignIpv6AddressOnCreation *bool
}

type Subnets []*Subnet

func (s *Subnet) setSubnet(src *ec2.Subnet) {
	s.VPCId = src.VpcId
	s.Tags = &Tags{}
	s.Tags.setTags(src.Tags)
	s.MapPublicIpOnLaunch = src.MapPublicIpOnLaunch
	s.CIDRBlock = src.CidrBlock
	s.AvailabilityZone = src.AvailabilityZone
	s.SubnetId = src.SubnetId
	s.AssignIpv6AddressOnCreation = src.AssignIpv6AddressOnCreation
	if len(src.Ipv6CidrBlockAssociationSet) > 0 {
		s.IPv6CIDRBlock = src.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock
	}
}

func (c *AWSClient) GetSubnets() (*Subnets, error) {
	data, err := c.ec2conn.DescribeSubnets(&ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, err
	}

	var output Subnets
	for _, v := range data.Subnets {
		tmp := &Subnet{}
		tmp.setSubnet(v)
		output = append(output, tmp)
	}

	return &output, nil
}

func (s *Subnets) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{}

	tmpl := `
	{{ if . }}
		{{- range . }}
	resource "aws_subnet" "{{ .SubnetId}}" {
    vpc_id = "{{ .VPCId}}"

    {{- if .AvailabilityZone}}
    availability_zone = "{{ .AvailabilityZone}}"
    {{- end }}

    {{- if .CIDRBlock}}
    cidr_block = "{{ .CIDRBlock}}"
    {{- end }}

    {{- if .IPv6CIDRBlock}}
    ipv6_cidr_block = "{{ .IPv6CIDRBlock }}"
    {{- end }}

    {{- if .MapPublicIpOnLaunch}}
    map_public_ip_on_launch = {{ .MapPublicIpOnLaunch }}
    {{- end }}

    {{- if .AssignIpv6AddressOnCreation}}
    assign_ipv6_address_on_creation = {{ .AssignIpv6AddressOnCreation}}
    {{- end}}

    {{- if .Tags}}
    tags {
      {{- range $k, $v := .Tags}}
      "{{ $k }}" = "{{ $v }}"
      {{- end}}
    }
    {{- end}}

	}
		{{- end}}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, s)
}

//**************** Security Group ****************
type SecurityGroup struct {
	Name        *string
	Description *string
	GroupId     *string
	Tags        *Tags
	VPCId       *string
	Ingresses   []*SecurityGroupRule
	Egresses    []*SecurityGroupRule
}

type SecurityGroups []*SecurityGroup

type SecurityGroupRule struct {
	FromPort             *int64
	ToPort               *int64
	IpProtocol           *string
	PrefixListIds        []*string
	CIDRBlocks           []*string
	IPv6CIDRBlock        []*string
	SourceSecurityGroups []*string
}

func (sg *SecurityGroup) setSecurityGroup(src *ec2.SecurityGroup, AccountId *string) {
	sg.Name = src.GroupName
	sg.Description = src.Description
	sg.GroupId = src.GroupId
	sg.Tags = &Tags{}
	sg.Tags.setTags(src.Tags)
	sg.VPCId = src.VpcId

	for _, v := range src.IpPermissions {
		var tmp SecurityGroupRule
		tmp.setRule(v, AccountId)
		sg.Ingresses = append(sg.Ingresses, &tmp)
	}

	for _, v := range src.IpPermissionsEgress {
		var tmp SecurityGroupRule
		tmp.setRule(v, AccountId)
		sg.Egresses = append(sg.Egresses, &tmp)
	}
}

func (r *SecurityGroupRule) setRule(src *ec2.IpPermission, AccountId *string) {
	r.FromPort = src.FromPort
	r.ToPort = src.ToPort
	r.IpProtocol = src.IpProtocol
	for _, v := range src.PrefixListIds {
		r.PrefixListIds = append(r.PrefixListIds, v.PrefixListId)
	}

	for _, v := range src.IpRanges {
		r.CIDRBlocks = append(r.CIDRBlocks, v.CidrIp)
	}

	for _, v := range src.Ipv6Ranges {
		r.IPv6CIDRBlock = append(r.IPv6CIDRBlock, v.CidrIpv6)
	}

	for _, v := range src.UserIdGroupPairs {
		if strings.Compare(*v.UserId, *AccountId) == 0 {
			r.SourceSecurityGroups = append(r.SourceSecurityGroups, v.GroupId)
		} else {
			srcSecGroup := fmt.Sprintf("%s/%s", *v.UserId, *v.GroupId)
			r.SourceSecurityGroups = append(r.SourceSecurityGroups, &srcSecGroup)
		}
	}
}

func (c *AWSClient) GetSecurityGroups(AccountId *string) (*SecurityGroups, error) {
	opt := ec2.DescribeSecurityGroupsInput{}
	var output SecurityGroups

	for {
		data, err := c.ec2conn.DescribeSecurityGroups(&opt)
		if err != nil {
			return nil, err
		}

		for _, v := range data.SecurityGroups {
			tmp := SecurityGroup{}
			tmp.setSecurityGroup(v, AccountId)
			output = append([]*SecurityGroup(output), &tmp)
		}

		if data.NextToken != nil {
			opt.NextToken = data.NextToken
		} else {
			break
		}
	}

	return &output, nil
}

func (sg *SecurityGroups) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"makeTerraformResourceName": makeTerraformResourceName,
		"joinStringSlice":           joinStringSlice,
		"StringValueSlice":          aws.StringValueSlice,
	}

	tmpl := `
	{{ if . }}
		{{- range . }}
	resource "aws_security_group" "{{ makeTerraformResourceName .Name }}" {
    name = "{{ .Name }}"

    {{- if .Description }}
    description = "{{ .Description }}"
    {{- end}}

    {{- if .VPCId}}
    vpc_id = "{{ .VPCId }}"
    {{- end}}

    {{- if .Tags }}
    tags {
      {{- range $k, $v := .Tags }}
      "{{ $k }}" = "{{ $v }}"
      {{- end }}
    }
    {{- end }}

    {{- if .Ingresses }}
      {{- range $k, $v := .Ingresses }}
      ingress {
        {{- if $v.FromPort }}
        from_port = "{{ $v.FromPort }}"
        {{- else }}
        from_port = 0
        {{- end }}

        {{- if $v.ToPort }}
        to_port = "{{ $v.ToPort }}"
        {{- else }}
        to_port = 0
        {{- end }}

        protocol = "{{ $v.IpProtocol }}"
        {{- if $v.PrefixListIds }}
        {{- $prefixlistIds := StringValueSlice $v.PrefixListIds}}
        prefix_list_ids = [{{ $prefixlistIds | joinStringSlice "," }}]
        {{- end}}

        {{- if $v.CIDRBlocks }}
        {{- $cidrblocks := StringValueSlice $v.CIDRBlocks }}
        cidr_blocks = [{{ $cidrblocks | joinStringSlice "," }}]
        {{- end }}

        {{- if $v.IPv6CIDRBlock }}
        {{- $ipv6cidr := StringValueSlice $v.IPv6CIDRBlock }}
        ipv6_cidr_blocks = [{{ $ipv6cidr | joinStringSlice ","}}]
        {{- end }}

        {{- if $v.SourceSecurityGroups }}
        {{- $src_secgroup := StringValueSlice $v.SourceSecurityGroups }}
        security_groups = [{{ $src_secgroup | joinStringSlice "," }}]
        {{- end }}
      }
      {{- end }}
    {{- end}}

    {{- if .Egresses }}
      {{- range $k, $v := .Egresses }}
      egress {
        {{- if $v.FromPort }}
        from_port = "{{ $v.FromPort }}"
        {{- else }}
        from_port = 0
        {{- end }}

        {{- if $v.ToPort }}
        to_port = "{{ $v.ToPort }}"
        {{- else }}
        to_port = 0
        {{- end }}

        protocol = "{{ $v.IpProtocol }}"
        {{- if $v.PrefixListIds }}
        {{- $prefixlistIds := StringValueSlice $v.PrefixListIds}}
        prefix_list_ids = [{{ $prefixlistIds | joinStringSlice "," }}]
        {{- end}}

        {{- if $v.SourceSecurityGroups }}
        {{- $src_secgroup := StringValueSlice $v.SourceSecurityGroups }}
        security_groups = [{{ $src_secgroup | joinStringSlice "," }}]
        {{- end }}

        {{- if $v.CIDRBlocks }}
        {{- $cidrblocks := StringValueSlice $v.CIDRBlocks }}
        cidr_blocks = [{{ $cidrblocks | joinStringSlice "," }}]
        {{- end }}

        {{- if $v.IPv6CIDRBlock }}
        {{- $ipv6cidr := StringValueSlice $v.IPv6CIDRBlock }}
        ipv6_cidr_blocks = [{{ $ipv6cidr | joinStringSlice ","}}]
        {{- end }}
      }
      {{- end }}
    {{- end}}

	}
		{{- end}}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, sg)
}
