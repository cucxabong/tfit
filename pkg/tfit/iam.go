package tfit

import (
	"io"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

type Policy struct {
	Description      *string
	Arn              *string
	Path             *string
	PolicyName       *string
	Document         *string
	DefaultVersionId *string
}

type Policies []*Policy

func (c *AWSClient) GetPolicy(p *Policy) error {
	out, err := c.iamconn.GetPolicy(&iam.GetPolicyInput{PolicyArn: p.Arn})
	if err != nil {
		return err
	}
	p.Description = out.Policy.Description
	p.DefaultVersionId = out.Policy.DefaultVersionId
	p.Path = out.Policy.Path
	p.PolicyName = out.Policy.PolicyName

	return nil
}

func (c *AWSClient) GetPolicyDocument(p *Policy) error {
	doc, err := c.iamconn.GetPolicyVersion(&iam.GetPolicyVersionInput{PolicyArn: p.Arn, VersionId: p.DefaultVersionId})
	if err != nil {
		return err
	}

	d, err := unEscapeHTML(doc.PolicyVersion.Document)
	if err != nil {
		return err
	}

	p.Document = &d
	return nil
}

// GetPolicies ...
func (c *AWSClient) GetPolicies() (*Policies, error) {
	var res Policies

	opt := &iam.ListPoliciesInput{
		Scope: aws.String(iam.PolicyScopeTypeLocal),
	}

	for {
		// Get all Local managed policies
		out, err := c.iamconn.ListPolicies(opt)
		if err != nil {
			return nil, err
		}

		ch := make(chan *chanItem, len(out.Policies))

		for _, v := range out.Policies {

			go func(Arn *string) {
				p := &Policy{
					Arn: Arn,
				}
				err := c.GetPolicy(p)
				if err != nil {
					ch <- &chanItem{err: err}
				}

				err = c.GetPolicyDocument(p)
				if err != nil {
					ch <- &chanItem{err: err}
				}

				ch <- &chanItem{obj: p}

			}(v.Arn)
		}

		for range out.Policies {
			receiver := <-ch
			if receiver.err != nil {
				return nil, err
			}

			res = append(res, receiver.obj.(*Policy))
		}

		// Check if output was truncated
		if aws.BoolValue(out.IsTruncated) {
			opt.Marker = out.Marker
		} else {
			break
		}
	}

	return &res, nil
}

// Render will render terraform format from 'Instances'
func (p *Policies) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		//"joinstring":       joinStringSlice,
		//"StringValueSlice": aws.StringValueSlice,
	}

	tmpl := `
	{{ if . }}
    {{ range . }}
    resource "aws_iam_policy" "{{ .PolicyName }}" {
      name = "{{ .PolicyName }}"
      {{- if .Path }}
      path = "{{.Path }}"
      {{- end }}
      {{- if .Description }}
      description = "{{ .Description }}"
      {{- end }}
      policy = <<EOF
      {{ .Document }}
EOF
    }
    {{- end }}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, p)
}
