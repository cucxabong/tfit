package tfit

import (
	"io"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

//**************** AutoScaling Group ****************

type TagDescription struct {
	Key               *string
	Value             *string
	PropagateAtLaunch *bool
}

type Group struct {
	AutoScalingGroupARN    *string
	Name                   *string
	MaxSize                *int64
	MinSize                *int64
	HealthCheckGracePeriod *int64
	HealthCheckType        *string
	DesiredCapacity        *int64
	DefaultCooldown        *int64

	PlacementGroup          *string
	LaunchConfigurationName *string
	LaunchTemplateName      *string

	AvailabilityZones []*string
	VPCZoneIdentifier []*string

	Tags                 []*TagDescription
	TerminationPolicies  []*string
	TargetGroupARNs      []*string
	EnabledMetrics       []*string
	ServiceLinkedRoleARN *string
}

type AutoScalingGroups []*Group

func (g *Group) parseVPCZoneIdentifier(src *string) {
	if src != nil && aws.StringValue(src) != "" {
		tok := strings.Split(aws.StringValue(src), ",")
		g.VPCZoneIdentifier = make([]*string, len(tok))
		for i := range tok {
			g.VPCZoneIdentifier[i] = aws.String(tok[i])
		}
	}
}

func (g *Group) setTags(src []*autoscaling.TagDescription) {
	g.Tags = make([]*TagDescription, len(src))

	for i, v := range src {
		g.Tags[i] = &TagDescription{Key: v.Key, Value: v.Value, PropagateAtLaunch: v.PropagateAtLaunch}
	}
}

func (g *Group) setEnabledMetrics(src []*autoscaling.EnabledMetric) {
	g.EnabledMetrics = make([]*string, len(src))
	for i, v := range src {
		g.EnabledMetrics[i] = v.Metric
	}
}

func (g *Group) setLaunchTemplateName(src *autoscaling.LaunchTemplateSpecification) {
	if src != nil {
		g.LaunchTemplateName = src.LaunchTemplateName
	}
}

func (g *Group) set(src *autoscaling.Group) {
	g.Name = src.AutoScalingGroupName
	g.MaxSize = src.MaxSize
	g.MinSize = src.MinSize
	g.HealthCheckGracePeriod = src.HealthCheckGracePeriod
	g.HealthCheckType = src.HealthCheckType
	g.DesiredCapacity = src.DesiredCapacity
	g.DefaultCooldown = src.DefaultCooldown
	g.PlacementGroup = src.PlacementGroup
	g.LaunchConfigurationName = src.LaunchConfigurationName
	g.AvailabilityZones = src.AvailabilityZones
	g.TerminationPolicies = src.TerminationPolicies
	g.TargetGroupARNs = src.TargetGroupARNs
	g.ServiceLinkedRoleARN = src.ServiceLinkedRoleARN
	g.setTags(src.Tags)
	g.setEnabledMetrics(src.EnabledMetrics)
	g.setLaunchTemplateName(src.LaunchTemplate)
	g.parseVPCZoneIdentifier(src.VPCZoneIdentifier)
}

// GetAutoScalingGroups craps a list of autoscaling group
// and placing it into a slice of 'AutoScalingGroups'
func (c *AWSClient) GetAutoScalingGroups() (*AutoScalingGroups, error) {
	var res AutoScalingGroups
	options := &autoscaling.DescribeAutoScalingGroupsInput{
		MaxRecords: aws.Int64(100),
	}

	for {
		groups, err := c.asconn.DescribeAutoScalingGroups(options)
		if err != nil {
			return nil, err
		}

		for _, v := range groups.AutoScalingGroups {
			tmp := &Group{}
			tmp.set(v)
			res = append(res, tmp)
		}

		if aws.StringValue(groups.NextToken) != "" {
			options.NextToken = groups.NextToken
		} else {
			break
		}
	}

	return &res, nil
}

// WriteHCL render terraform configs from AutoScalingGroups
// and pretty print int into io.Writer
func (src *AutoScalingGroups) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"StringValueSlice": aws.StringValueSlice,
	}

	tmpl := `
    {{- if .}}
    {{- range .}}
    resource "aws_autoscaling_group" "{{ .Name }}" {
      name = "{{ .Name }}"
      min_size = {{ .MinSize }}
      max_size = {{ .MaxSize }}
      {{- if .HealthCheckGracePeriod }}
      health_check_grace_period = {{ .HealthCheckGracePeriod }}
      {{- end }}
      {{- if .HealthCheckType }}
      health_check_type = "{{ .HealthCheckType }}"
      {{- end }}
      {{- if .DesiredCapacity }}
      desired_capacity = {{ .DesiredCapacity }}
      {{- end }}
      {{- if .DefaultCooldown }}
      default_cooldown = {{ .DefaultCooldown }}
      {{- end }}
      {{- if .PlacementGroup }}
      placement_group  = "{{ .PlacementGroup }}"
      {{- end }}
      {{- if .LaunchConfigurationName }}
      launch_configuration = "{{ .LaunchConfigurationName }}"
      {{- end }}
      {{- if .LaunchTemplateName}}
      launch_template = "{{ .LaunchTemplateName }}"
      {{- end }}
      {{- if .ServiceLinkedRoleARN }}
      service_linked_role_arn = "{{ .ServiceLinkedRoleARN }}"
      {{- end }}
      {{- if .Tags }}
      tags = [
        {{ range .Tags }}
        {
          key = "{{ .Key }}"
          value = "{{ .Value }}"
          propagate_at_launch = {{ .PropagateAtLaunch }}
        },
        {{ end }}
        ]
      {{- end }}
      {{- if .VPCZoneIdentifier }}
      {{ $zids := StringValueSlice .VPCZoneIdentifier }}
      vpc_zone_identifier = [{{ $zids | joinstring "," }}]
      {{- end }}
      {{- if .AvailabilityZones }}
      {{ $zones := StringValueSlice .AvailabilityZones }}
      availability_zones = [{{ $zones | joinstring "," }}]
      {{end}}
      {{ if .TerminationPolicies }}
      {{ $policies := StringValueSlice .TerminationPolicies }}
      termination_policies = [{{ $policies | joinstring "," }}]
      {{ end }}
      {{ if .TargetGroupARNs }}
      {{ $tgs := StringValueSlice .TargetGroupARNs }}
      target_group_arns = [{{ $tgs | joinstring "," }}]
      {{end}}
      {{ if .EnabledMetrics }}
      {{ $metrics := StringValueSlice .EnabledMetrics }}
      enabled_metrics = [{{ $metrics | joinstring "," }}]
      {{end}}
    }
    {{- end}}
  {{- end }}
  `
	return renderHCL(w, tmpl, funcMap, src)
}

//**************** Launch Configuration ****************
type LaunchConfigurations []*autoscaling.LaunchConfiguration

func (c *AWSClient) GetLaunchConfigurations() (*LaunchConfigurations, error) {
	var res LaunchConfigurations

	options := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}

	for {
		launchconfigs, err := c.asconn.DescribeLaunchConfigurations(options)
		if err != nil {
			return nil, err
		}

		res = append(res, launchconfigs.LaunchConfigurations...)

		if aws.StringValue(launchconfigs.NextToken) != "" {
			options.NextToken = launchconfigs.NextToken
		} else {
			break
		}

	}
	return &res, nil
}

func (src *LaunchConfigurations) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"StringValueSlice": aws.StringValueSlice,
	}

	tmpl := `
  {{- if . }}
    {{- range .}}
    resource "aws_launch_configuration" "{{ .LaunchConfigurationName }}" {
      name = "{{ .LaunchConfigurationName }}"
      image_id = "{{ .ImageId }}"
      instance_type = "{{ .InstanceType }}"
      {{- if .IamInstanceProfile }}
      iam_instance_profile = "{{ .IamInstanceProfile }}"
      {{- end}}
      {{- if .KeyName }}
      key_name = "{{ .KeyName }}"
      {{- end}}
      {{- if .AssociatePublicIpAddress }}
      associate_public_ip_address = {{ .AssociatePublicIpAddress }}
      {{- end}}
      {{- if .ClassicLinkVPCId }}
      vpc_classic_link_id = "{{ .ClassicLinkVPCId }}"
      {{- end }}
      {{- if .ClassicLinkVPCSecurityGroups}}
      {{- $classicSecGroup := StringValueSlice .ClassicLinkVPCSecurityGroups}}
      vpc_classic_link_security_groups = [{{ $classicSecGroup | joinstring "," }}]
      {{- end}}
      {{- $l := len .UserData }}
      {{- $notempty := gt $l 0}}
      {{- if and .UserData $notempty }}
      user_data = "{{ .UserData }}"
      {{- end}}
      {{- if .InstanceMonitoring }}
      enable_monitoring = {{ .InstanceMonitoring.Enabled }}
      {{- end}}
      ebs_optimized = {{ .EbsOptimized }}
      {{- if .PlacementTenancy}}
      placement_tenancy = "{{ .PlacementTenancy }}"
      {{- end}}
      {{- if .SecurityGroups }}
      {{- $secgroups := StringValueSlice .SecurityGroups}}
      security_groups = [{{ $secgroups | joinstring "," }}]
      {{- end}}
      {{- if .BlockDeviceMappings }}
        {{- range .BlockDeviceMappings}}
          {{- if .VirtualName}}
          ephemeral_block_device {
            device_name = "{{ .DeviceName }}"
            virtual_name = "{{ .VirtualName }}"
          }
          {{else if .NoDevice }}
          root_block_device {
            {{- if .Ebs.VolumeType}}
            volume_type = "{{ .Ebs.VolumeType }}"
            {{end}}
            {{- if .Ebs.VolumeSize}}
            volume_size = {{ .Ebs.VolumeSize }}
            {{end}}
            {{- if .Ebs.Iops}}
            iops = {{ .Ebs.Iops }}
            {{end}}
            {{- if .Ebs.DeleteOnTermination}}
            delete_on_termination = {{ .Ebs.DeleteOnTermination }}
            {{end}}
          }
          {{else }}
          ebs_block_device {
            device_name = "{{ .DeviceName }}"
            {{- if .Ebs.SnapshotId }}
            snapshot_id = "{{ .Ebs.SnapshotId }}"
            {{- end}}
            {{- if .Ebs.VolumeType}}
            volume_type = "{{ .Ebs.VolumeType }}"
            {{- end}}
            {{- if .Ebs.VolumeSize}}
            volume_size = {{ .Ebs.VolumeSize }}
            {{- end}}
            {{- if .Ebs.Iops}}
            iops = {{ .Ebs.Iops }}
            {{- end}}
            {{- if .Ebs.DeleteOnTermination}}
            delete_on_termination = {{ .Ebs.DeleteOnTermination }}
            {{- end}}
            {{- if .Ebs.Encrypted }}
            encrypted = {{ .Ebs.Encrypted }}
            {{- end}}
          }
          {{end}}
        {{end}}
      {{end}}
    }
    {{end}}
  {{end}}
  `
	return renderHCL(w, tmpl, funcMap, src)
}
