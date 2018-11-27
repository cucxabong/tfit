package tfit

import (
	"fmt"
	"io"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

//**************** IAM Customer Managed Policy ****************

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

func (p *Policies) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{}

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

//**************** IAM Role ****************
type Role struct {
	Name                     *string
	AssumeRolePolicyDocument *string
	RoleId                   *string
	Description              *string
	Path                     *string
	MaxSessionDuration       *int64
	PermissionBoundaryArn    *string
}

type Roles []*Role

func (c *AWSClient) ListRoles() (*Roles, error) {
	opt := iam.ListRolesInput{}
	var output Roles
	for {
		data, err := c.iamconn.ListRoles(&opt)
		if err != nil {
			return nil, err
		}

		for _, v := range data.Roles {
			tmp := Role{
				AssumeRolePolicyDocument: v.AssumeRolePolicyDocument,
				Description:              v.Description,
				Path:                     v.Path,
				MaxSessionDuration:       v.MaxSessionDuration,
				RoleId:                   v.RoleId,
				Name:                     v.RoleName,
			}
			if v.PermissionsBoundary != nil {
				tmp.PermissionBoundaryArn = v.PermissionsBoundary.PermissionsBoundaryArn
			}

			unEscapeAssumeRole, err := unEscapeHTML(tmp.AssumeRolePolicyDocument)
			if err != nil {
				fmt.Println(err)
				continue
			} else {
				tmp.AssumeRolePolicyDocument = &unEscapeAssumeRole
			}

			output = append(output, &tmp)
		}

		if data.IsTruncated != nil && aws.BoolValue(data.IsTruncated) {
			opt.Marker = data.Marker
		} else {
			break
		}
	}

	return &output, nil
}

func (r *Roles) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"makeTerraformResourceName": makeTerraformResourceName,
		"prettyJSON":                prettyJSON,
	}

	tmpl := `
	{{ if . }}
    {{ range . }}
    resource "aws_iam_policy" "{{ .Name | makeTerraformResourceName }}" {
      name = "{{ .Name }}"
      assume_role_policy = <<EOF
      {{ .AssumeRolePolicyDocument | prettyJSON }}
EOF
      {{- if .Path }}
      path = "{{ .Path }}"
      {{- end }}

      {{- if .Description }}
      description = "{{ .Description }}"
      {{- end }}

      {{- if .MaxSessionDuration }}
      max_session_duration = {{.MaxSessionDuration}}
      {{- end }}

      {{- if .PermissionBoundaryArn}}
      permissions_boundary = "{{ .PermissionBoundaryArn }}"
      {{- end }}
    }
    {{- end }}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, r)
}

//**************** IAM User ****************
type User struct {
	Path                   *string
	Tags                   *Tags
	UserId                 *string
	UserName               *string
	PermissionsBoundaryArn *string
}

func (u *User) setUser(src *iam.User) {
	u.Path = src.Path
	u.Tags = &Tags{}
	for _, v := range src.Tags {
		map[string]*string(*u.Tags)[*v.Key] = v.Value
	}
	u.UserId = src.UserId
	u.UserName = src.UserName
	if src.PermissionsBoundary != nil {
		u.PermissionsBoundaryArn = src.PermissionsBoundary.PermissionsBoundaryArn
	}
}

type Users []*User

func (c *AWSClient) ListUsers() (*Users, error) {
	opt := iam.ListUsersInput{}

	var output Users
	for {
		data, err := c.iamconn.ListUsers(&opt)
		if err != nil {
			return nil, err
		}
		for _, v := range data.Users {
			var u User
			u.setUser(v)
			output = append(output, &u)
		}

		if data.IsTruncated != nil && aws.BoolValue(data.IsTruncated) {
			opt.Marker = data.Marker
		} else {
			break
		}
	}

	return &output, nil
}

func (r *Users) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"makeTerraformResourceName": makeTerraformResourceName,
	}

	tmpl := `
	{{ if . }}
    {{ range . }}
    resource "aws_iam_user" "{{ .UserName | makeTerraformResourceName }}" {
      name = "{{ .UserName }}"
      {{- if .Path }}
      path = "{{ .Path }}"
      {{- end}}

      {{- if .PermissionsBoundaryArn }}
      permissions_boundary = "{{ .PermissionsBoundaryArn }}"
      {{- end }}

      {{- if gt (len .Tags) 0 }}
      tags {
        {{- range $k, $v := .Tags }}
        "{{ $k }}" = "{{ $v }}"
        {{- end}}
      }
      {{- end }}
    }
    {{- end }}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, r)
}

//**************** IAM Group ****************
type IAMGroup struct {
	Name *string
	Id   *string
	Path *string
}

type IAMGroups []*IAMGroup

func (c *AWSClient) ListIAMGroups() (*IAMGroups, error) {
	opt := iam.ListGroupsInput{}
	var output IAMGroups
	for {
		data, err := c.iamconn.ListGroups(&opt)
		if err != nil {
			return nil, err
		}

		for _, g := range data.Groups {
			tmp := IAMGroup{
				Name: g.GroupName,
				Id:   g.GroupId,
				Path: g.Path,
			}
			output = append(output, &tmp)
		}

		if data.IsTruncated != nil && aws.BoolValue(data.IsTruncated) {
			opt.Marker = data.Marker
		} else {
			break
		}
	}

	return &output, nil
}

func (g *IAMGroups) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"makeTerraformResourceName": makeTerraformResourceName,
	}

	tmpl := `
	{{ if . }}
    {{ range . }}
    resource "aws_iam_group" "{{ .Name | makeTerraformResourceName }}" {
      name = "{{ .Name }}"
      {{- if .Path }}
      path = "{{ .Path }}"
      {{- end }}
    }
    {{- end }}
	{{- end}}
	`
	return renderHCL(w, tmpl, funcMap, g)
}
