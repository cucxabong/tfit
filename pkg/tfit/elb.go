package tfit

import (
	"io"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
)

type ELBHealthCheck struct {
	HealthyThreshold   *int64
	UnhealthyThreshold *int64
	Timeout            *int64
	Interval           *int64
	Target             *string
}

type ELBListerner struct {
	InstancePort         *int64
	InstanceProtocol     *string
	LoadBalancerPort     *int64
	LoadBalancerProtocol *string
	SSLCertificateId     *string
}

type ELBAccessLog struct {
	Enabled        *bool
	S3BucketName   *string
	S3BucketPrefix *string
	EmitInterval   *int64
}

type ELB struct {
	Name                      *string
	AvailabilityZones         []*string
	SecurityGroups            []*string
	Subnets                   []*string
	Instances                 []*string
	HealthCheck               *ELBHealthCheck
	Internal                  *bool
	Listeners                 []*ELBListerner
	AccessLog                 *ELBAccessLog
	CrossZoneLoadBalancing    *bool
	ConnectionDraining        *bool
	ConnectionDrainingTimeOut *int64
	IdleTimeout               *int64

	Tags map[string]*string
}

type ELBs []*ELB

func (e *ELB) setInstances(src []*elb.Instance) {
	if src != nil {
		for _, v := range src {
			e.Instances = append(e.Instances, v.InstanceId)
		}
	}
}

func (e *ELB) setListener(src []*elb.ListenerDescription) {
	if src != nil {
		for _, v := range src {
			tmp := ELBListerner{
				InstancePort:         v.Listener.InstancePort,
				InstanceProtocol:     v.Listener.InstanceProtocol,
				LoadBalancerPort:     v.Listener.LoadBalancerPort,
				LoadBalancerProtocol: v.Listener.Protocol,
				SSLCertificateId:     v.Listener.SSLCertificateId,
			}

			e.Listeners = append(e.Listeners, &tmp)
		}
	}
}

func (e *ELB) setHealthCheck(src *elb.HealthCheck) {
	if src != nil {
		tmp := &ELBHealthCheck{
			HealthyThreshold:   src.HealthyThreshold,
			UnhealthyThreshold: src.UnhealthyThreshold,
			Timeout:            src.Timeout,
			Interval:           src.Interval,
			Target:             src.Target,
		}

		e.HealthCheck = tmp
	}
}

func (e *ELB) setAccessLog(src *elb.AccessLog) {
	if src != nil && aws.BoolValue(src.Enabled) {
		tmp := ELBAccessLog{
			Enabled: src.Enabled,
		}
		if src.EmitInterval != nil {
			tmp.EmitInterval = src.EmitInterval
		}
		if src.S3BucketName != nil {
			tmp.S3BucketName = src.S3BucketName
		}
		if src.S3BucketPrefix != nil {
			tmp.S3BucketPrefix = src.S3BucketPrefix
		}
		e.AccessLog = &tmp
	}
}

func (e *ELB) setELBAttributes(src *elb.LoadBalancerDescription, c *AWSClient) error {
	e.setInstances(src.Instances)
	e.setHealthCheck(src.HealthCheck)

	if src.Scheme != nil {
		if strings.Compare(aws.StringValue(src.Scheme), "internal") == 0 {
			e.Internal = aws.Bool(true)
		} else {
			e.Internal = aws.Bool(false)
		}
	}
	e.setListener(src.ListenerDescriptions)

	opt := elb.DescribeLoadBalancerAttributesInput{LoadBalancerName: e.Name}
	data, err := c.elbconn.DescribeLoadBalancerAttributes(&opt)
	if err != nil {
		return err
	}
	e.setAccessLog(data.LoadBalancerAttributes.AccessLog)

	if data.LoadBalancerAttributes.ConnectionDraining != nil {
		e.ConnectionDraining = data.LoadBalancerAttributes.ConnectionDraining.Enabled
		e.ConnectionDrainingTimeOut = data.LoadBalancerAttributes.ConnectionDraining.Timeout
	}

	if data.LoadBalancerAttributes.ConnectionSettings != nil {
		e.IdleTimeout = data.LoadBalancerAttributes.ConnectionSettings.IdleTimeout
	}

	if data.LoadBalancerAttributes.CrossZoneLoadBalancing != nil {
		e.CrossZoneLoadBalancing = data.LoadBalancerAttributes.CrossZoneLoadBalancing.Enabled
	}

	// Get Tags
	describeTagsOpt := elb.DescribeTagsInput{
		LoadBalancerNames: []*string{e.Name},
	}
	tagsOutput, err := c.elbconn.DescribeTags(&describeTagsOpt)
	if err != nil {
		return err
	}

	if len(tagsOutput.TagDescriptions) > 0 && len(tagsOutput.TagDescriptions[0].Tags) > 0 {
		e.Tags = make(map[string]*string)
		for _, t := range tagsOutput.TagDescriptions[0].Tags {
			e.Tags[aws.StringValue(t.Key)] = t.Value
		}
	}

	return nil
}

func (c *AWSClient) ListELBs() (*ELBs, error) {
	opt := elb.DescribeLoadBalancersInput{}
	var output ELBs
	for {
		data, err := c.elbconn.DescribeLoadBalancers(&opt)
		if err != nil {
			return nil, err
		}

		for _, v := range data.LoadBalancerDescriptions {
			tmp := ELB{
				Name:              v.LoadBalancerName,
				AvailabilityZones: v.AvailabilityZones,
				SecurityGroups:    v.SecurityGroups,
				Subnets:           v.Subnets,
			}

			err := tmp.setELBAttributes(v, c)
			if err != nil {
				return nil, err
			}
			output = append(output, &tmp)

		}

		if data.NextMarker != nil {
			opt.Marker = data.NextMarker
		} else {
			break
		}
	}

	return &output, nil

}

func (elb *ELBs) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"StringValueSlice": aws.StringValueSlice,
	}

	tmpl := `
	{{ if . }}
		{{ range . }}
	resource "aws_elb" "{{ .Name }}" {
    name = "{{ .Name }}"

    {{- if .AvailabilityZones }}
    availability_zones = [{{ joinstring "," (StringValueSlice .AvailabilityZones)}}]
    {{- end }}

    {{- if .AccessLog }}
    access_logs {
      bucket = "{{ .AccessLog.S3BucketName}}"
      {{- if .AccessLog.Enabled }}
      enabled = {{ .AccessLog.Enabled }}
      {{- end }}

      {{- if .AccessLog.S3BucketPrefix }}
      bucket_prefix = "{{ .AccessLog.S3BucketPrefix }}"
      {{- end }}

      {{- if .AccessLog.EmitInterval}}
      interval  = {{  .AccessLog.EmitInterval }}
      {{- end }}
    }
    {{- end}}

    {{- if .SecurityGroups }}
    security_groups = [{{ joinstring "," (StringValueSlice .SecurityGroups )}}]
    {{- end }}

    {{- if .Subnets }}
    subnets = [{{ joinstring "," (StringValueSlice .Subnets )}}]
    {{- end }}

    {{- if .Instances }}
    instances = [{{ joinstring "," (StringValueSlice .Instances )}}]
    {{- end }}

    {{- if .Internal }}
    internal = {{ .Internal }}
    {{- end }}

    {{- if .CrossZoneLoadBalancing }}
    cross_zone_load_balancing = {{ .CrossZoneLoadBalancing }}
    {{- end }}

    {{- if .ConnectionDraining }}
    connection_draining = {{ .ConnectionDraining }}
    {{- end }}

    {{- if .ConnectionDrainingTimeOut }}
    connection_draining_timeout = {{ .ConnectionDrainingTimeOut }}
    {{- end }}

    {{- if .IdleTimeout }}
    idle_timeout = {{ .IdleTimeout }}
    {{- end }}

    {{- if .HealthCheck }}
    health_check {
      healthy_threshold = {{ .HealthCheck.HealthyThreshold}}
      unhealthy_threshold  = {{ .HealthCheck.UnhealthyThreshold}}
      target = "{{ .HealthCheck.Target}}"
      internal = {{ .HealthCheck.Interval}}
      timeout = {{ .HealthCheck.Timeout}}
    }
    {{- end }}

    {{- if .Listeners }}
      {{- range .Listeners }}
      listener {
        instance_port = {{ .InstancePort }}
        instance_protocol = "{{ .InstanceProtocol }}"
        lb_port = {{ .LoadBalancerPort }}
        lb_protocol = "{{ .LoadBalancerProtocol }}"
        {{- if .SSLCertificateId}}
        ssl_certificate_id = "{{ .SSLCertificateId }}"
        {{- end }}
      }
      {{- end }}
    {{- end }}

    {{- if .Tags }}
      tags {
        {{- range $k, $v := .Tags }}
        "{{ $k }}" = "{{ $v }}"
        {{- end }}
      }
    {{- end }}

	}
		{{- end}}
	{{- end}}
  `

	return renderHCL(w, tmpl, funcMap, elb)

}
