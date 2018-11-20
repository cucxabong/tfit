package tfit

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

type Route53Zone struct {
	Name            *string
	Comment         *string
	VPCId           *string
	VPCRegion       *string
	ZoneId          *string
	DelegationSetId *string
	NameServers     []*string
	Tags            map[*string]*string
}

type Zones []*Route53Zone

func (z *Route53Zone) set(data *route53.HostedZone) {
	z.Name = data.Name
	z.ZoneId = getZoneId(data.Id)

	if data.Config != nil && data.Config.Comment != nil {
		z.Comment = data.Config.Comment
	}
}

func (c *AWSClient) GetHostZones(maxRoutines int) (*Zones, error) {
	r53 := c.r53conn
	var res Zones
	opt := &route53.ListHostedZonesInput{}
	for {
		zones, err := r53.ListHostedZones(opt)
		if err != nil {
			return nil, err
		}

		if zones != nil {
			ch := make(chan *chanItem, len(zones.HostedZones))
			lock := make(chan struct{}, maxRoutines)

			for _, v := range zones.HostedZones {
				// Ignore Private hosted zone
				if *v.Config.PrivateZone {
					ch <- &chanItem{obj: nil, err: fmt.Errorf("Private Zone")}
				}

				// Get lock
				lock <- struct{}{}
				go func(v *route53.HostedZone) {
					z := &Route53Zone{}
					z.set(v)

					// Get tags
					req := &route53.ListTagsForResourceInput{
						ResourceId:   z.ZoneId,
						ResourceType: aws.String("hostedzone"),
					}

					resp, err := r53.ListTagsForResource(req)
					if err != nil {
						ch <- &chanItem{obj: nil, err: err}
					}
					z.Tags = make(map[*string]*string)
					if resp.ResourceTagSet != nil && resp.ResourceTagSet.Tags != nil {
						for i := range resp.ResourceTagSet.Tags {
							z.Tags[resp.ResourceTagSet.Tags[i].Key] = resp.ResourceTagSet.Tags[i].Value
						}
					}

					ch <- &chanItem{obj: z, err: nil}
					<-lock
				}(v)
			}

			for range zones.HostedZones {
				receiver := <-ch
				if receiver.err != nil {
					//return nil, receiver.err
					// [TODO] Handing error here
					continue
				}
				res = append(res, receiver.obj.(*Route53Zone))
			}
		}

		if zones.IsTruncated != nil && aws.BoolValue(zones.IsTruncated) {
			opt.Marker = zones.NextMarker
		} else {
			break
		}
	}

	return &res, nil
}

func (zs *Zones) WriteHCL(w io.Writer) error {

	tmpl := `
		{{ if . }}
      {{ range . }}
      {{- $resource_name := TrimSuffix .Name "." }}
				resource "aws_route53_zone" "{{ replace $resource_name "." "-" -1}}" {
					name = "{{ .Name }}"
          {{- if .Comment }}
          comment = "{{ .Comment }}"
          {{- end}}
          {{- if .Tags }}
          tags {
						{{- range $k, $v := .Tags}}
							"{{$k}}" = "{{$v}}"
            {{end}}
          }
					{{end}}
				}
			{{end}}
		{{end}}
	`
	funcMap := template.FuncMap{
		"replace":    strings.Replace,
		"TrimSuffix": strings.TrimSuffix,
	}

	return renderHCL(w, tmpl, funcMap, *zs)
}

func (z *Zones) WriteTerraformImportCmd(w io.Writer) error {
	funcMap := template.FuncMap{
		"replace":    strings.Replace,
		"TrimSuffix": strings.TrimSuffix,
	}

	tmpl := `
  {{if .}}
    {{- range .}}
    {{- $resource_name := TrimSuffix .Name "." }}
terraform import aws_route53_zone.{{ replace $resource_name "." "-" -1}} {{.ZoneId}}
    {{- end}}
  {{- end}}
  `

	return renderTerraformImportCmd(w, tmpl, funcMap, z)
}

type RecordAlias struct {
	Name                 *string
	ZoneId               *string
	EvaluateTargetHealth *bool
}

type RecordSet struct {
	Name    *string
	ZoneId  *string
	Type    *string
	TTL     *int64
	Records []*string
	Alias   *RecordAlias
}

type RecordSets []RecordSet

func (r *RecordSet) setAlias(src *route53.AliasTarget) {
	r.Alias = &RecordAlias{}
	r.Alias.ZoneId = src.HostedZoneId
	r.Alias.Name = src.DNSName
	r.Alias.EvaluateTargetHealth = src.EvaluateTargetHealth
}

func (r *RecordSet) setValue(src []*route53.ResourceRecord) {
	for _, v := range src {
		r.Records = append(r.Records, v.Value)
	}
}

func (c *AWSClient) GetResourceRecordSets(ZoneId *string) (*RecordSets, error) {
	r53 := c.r53conn
	results := RecordSets{}

	opt := &route53.ListResourceRecordSetsInput{HostedZoneId: ZoneId}
	for {
		records, err := r53.ListResourceRecordSets(opt)
		if err != nil {
			return nil, err
		}

		for _, v := range records.ResourceRecordSets {
			r := RecordSet{}
			if v.AliasTarget != nil {
				r.setAlias(v.AliasTarget)
			}

			if v.ResourceRecords != nil {
				r.setValue(v.ResourceRecords)
			}

			r.ZoneId = getZoneId(ZoneId)
			r.Name = v.Name
			r.TTL = v.TTL
			r.Type = v.Type

			results = append(results, r)
		}

		if records.IsTruncated != nil && aws.BoolValue(records.IsTruncated) {
			opt.StartRecordName = records.NextRecordName
			opt.StartRecordType = records.NextRecordType
		} else {
			break
		}
	}
	return &results, nil
}

func (c *AWSClient) GetAllResourceRecordSets() (*RecordSets, error) {
	// Get all hosted zones
	zones, err := c.GetHostZones(5)
	results := RecordSets{}

	if err != nil {
		return nil, err
	}

	for _, v := range []*Route53Zone(*zones) {
		//		prettyJson, err := url.QueryUnescape(aws.StringValue([]*Policy(*polices)[0].Document))
		zId := v.ZoneId
		r, err := c.GetResourceRecordSets(zId)
		if err != nil {
			return nil, err
		}
		results = append(results, []RecordSet(*r)...)
	}

	return &results, nil

}

func (rs *RecordSets) WriteTerraformImportCmd(w io.Writer) error {
	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"replace":          strings.Replace,
		"int64":            aws.Int64Value,
		"StringValueSlice": aws.StringValueSlice,
		"TrimSuffix":       strings.TrimSuffix,
	}

	tmpl := `
  {{if .}}
    {{range .}}
    {{- $resource_name := TrimSuffix .Name "." }}
terraform import aws_route53_record.{{ replace $resource_name "." "_" -1 }}-{{.Type}} {{.ZoneId}}_{{TrimSuffix .Name "."}}_{{.Type}}
    {{- end}}
  {{- end}}
  `
	return renderTerraformImportCmd(w, tmpl, funcMap, rs)

}

func (rs *RecordSets) WriteHCL(w io.Writer) error {

	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"replace":          strings.Replace,
		"int64":            aws.Int64Value,
		"StringValueSlice": aws.StringValueSlice,
		"TrimSuffix":       strings.TrimSuffix,
	}

	tmpl := `
	{{if . }}
    {{ range . }}
    {{- $resource_name := TrimSuffix .Name "." }}
			resource "aws_route53_record" "{{ replace $resource_name "." "_" -1 }}-{{.Type}}" {
				zone_id = "{{ .ZoneId }}"
				name = "{{.Name}}"
        type = "{{.Type}}"
        {{- $ttl := int64 .TTL}}
				{{if gt $ttl 0 }}
				ttl = {{.TTL}}
				{{end}}
        {{ if .Records }}
        {{- $rc := StringValueSlice .Records}}
				records = [{{ $rc | joinstring  "," }}]
				{{end}}
				{{if .Alias }}
				alias {
					name = "{{ .Alias.Name }}"
					zone_id = "{{ .Alias.ZoneId }}"
					evaluate_target_health = {{.Alias.EvaluateTargetHealth}}
				}
				{{end}}
			}
		{{end}}
	{{end}}
	`

	return renderHCL(w, tmpl, funcMap, rs)

}
