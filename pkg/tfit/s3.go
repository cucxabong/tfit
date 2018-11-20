package tfit

import (
	"io"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3LifecycleRule struct {
	ID                           *string
	Enable                       *bool
	Prefix                       *string
	Transition                   []*s3.Transition
	NoncurrentVersionTransitions []*s3.NoncurrentVersionTransition
	NoncurrentVersionExpiration  *s3.NoncurrentVersionExpiration
}

type BucketVersioning struct {
	Enabled   *bool
	MFADelete *bool
}

type Bucket struct {
	Name                              *string
	Policy                            *string
	Website                           *s3.GetBucketWebsiteOutput
	LifecycleRules                    []*S3LifecycleRule
	ReplicationConfiguration          *s3.ReplicationConfiguration
	ServerSideEncryptionConfiguration *s3.ServerSideEncryptionConfiguration
	CORSRules                         []*s3.CORSRule
	Logging                           *s3.LoggingEnabled
	Versioning                        *BucketVersioning
}

type Buckets []*Bucket

func (b *Bucket) getBucketPoliy(c *AWSClient) error {
	output, err := c.s3conn.GetBucketPolicy(&s3.GetBucketPolicyInput{Bucket: b.Name})
	if err != nil {
		return handleError(err)
	}

	b.Policy = output.Policy

	return nil
}

func (b *Bucket) getWebsite(c *AWSClient) error {
	output, err := c.s3conn.GetBucketWebsite(&s3.GetBucketWebsiteInput{Bucket: b.Name})
	if err != nil {
		return handleError(err)
	}

	b.Website = output

	return nil
}

func (b *Bucket) setLifecycleRule(src []*s3.LifecycleRule) error {
	b.LifecycleRules = make([]*S3LifecycleRule, len(src))
	for i := range src {
		b.LifecycleRules[i] = &S3LifecycleRule{}
		b.LifecycleRules[i].ID = src[i].ID
		b.LifecycleRules[i].Prefix = src[i].Prefix
		b.LifecycleRules[i].Transition = src[i].Transitions
		b.LifecycleRules[i].NoncurrentVersionExpiration = src[i].NoncurrentVersionExpiration
		b.LifecycleRules[i].NoncurrentVersionTransitions = src[i].NoncurrentVersionTransitions
		if src[i].Status != nil && aws.StringValue(src[i].Status) == "Enabled" {
			b.LifecycleRules[i].Enable = aws.Bool(true)
		}
	}

	return nil
}

func (b *Bucket) getBucketLocation(c *AWSClient) (*string, error) {
	output, err := c.s3conn.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: b.Name})
	if err != nil {
		e := handleError(err)
		if e != nil {
			return nil, e
		} else {
			return nil, nil
		}
	}

	return output.LocationConstraint, nil

}

func (b *Bucket) getReplicationConfiguration(c *AWSClient) error {
	output, err := c.s3conn.GetBucketReplication(&s3.GetBucketReplicationInput{Bucket: b.Name})
	if err != nil {
		return handleError(err)
	}

	b.ReplicationConfiguration = output.ReplicationConfiguration
	return nil
}

func (b *Bucket) getLifecycleRules(c *AWSClient) error {
	output, err := c.s3conn.GetBucketLifecycleConfiguration(&s3.GetBucketLifecycleConfigurationInput{Bucket: b.Name})
	if err != nil {
		return handleError(err)
	}

	return b.setLifecycleRule(output.Rules)
}

func (b *Bucket) getServerSideEncryptionConfiguration(c *AWSClient) error {
	output, err := c.s3conn.GetBucketEncryption(&s3.GetBucketEncryptionInput{Bucket: b.Name})
	if err != nil {
		return handleError(err)
	}

	b.ServerSideEncryptionConfiguration = output.ServerSideEncryptionConfiguration

	return nil
}

func (b *Bucket) getLogging(c *AWSClient) error {
	output, err := c.s3conn.GetBucketLogging(&s3.GetBucketLoggingInput{Bucket: b.Name})
	if err != nil {
		return err
	}

	b.Logging = output.LoggingEnabled

	return nil
}

func (b *Bucket) getCORSRule(c *AWSClient) error {
	output, err := c.s3conn.GetBucketCors(&s3.GetBucketCorsInput{Bucket: b.Name})
	if err != nil {
		return handleError(err)
	}

	b.CORSRules = output.CORSRules
	return nil
}

func (b *Bucket) getVersioning(c *AWSClient) error {
	output, err := c.s3conn.GetBucketVersioning(&s3.GetBucketVersioningInput{Bucket: b.Name})
	if err != nil {
		return err
	}
	b.Versioning = &BucketVersioning{}
	if output.Status != nil && aws.StringValue(output.Status) == s3.BucketVersioningStatusEnabled {
		b.Versioning.Enabled = aws.Bool(true)
	}

	if output.MFADelete != nil && aws.StringValue(output.MFADelete) == s3.MFADeleteStatusEnabled {
		b.Versioning.MFADelete = aws.Bool(true)
	} else {
		b.Versioning.MFADelete = aws.Bool(false)
	}

	return nil
}

func (b *Bucket) GetBucketDetails(c *AWSClient) error {
	// Get Bucket Policy
	if err := b.getBucketPoliy(c); err != nil {
		return err
	}

	// Get Website detail
	if err := b.getWebsite(c); err != nil {
		return err
	}

	// Get Lifecycle Rules
	if err := b.getLifecycleRules(c); err != nil {
		return err
	}

	// Get Replication Configuration
	if err := b.getReplicationConfiguration(c); err != nil {
		return err
	}

	// Get Server Side Encryption
	if err := b.getServerSideEncryptionConfiguration(c); err != nil {
		return err
	}

	// Get Logging
	if err := b.getLogging(c); err != nil {
		return err
	}

	// Get CORS Rules
	if err := b.getCORSRule(c); err != nil {
		return err
	}

	// Get Versioning
	if err := b.getVersioning(c); err != nil {
		return err
	}

	return nil
}

func (c *AWSClient) GetBuckets() (*Buckets, error) {
	var res Buckets
	output, err := c.s3conn.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	/*
		bucketName := "internal.sw.prd.cost-management"
		bucket := &Bucket{Name: aws.String(bucketName)}
		if err := bucket.GetBucketDetails(c); err != nil {
			return nil, err
		}
		res = append(res, bucket)
	*/
	ch := make(chan *chanItem, len(output.Buckets))
	blk := make(chan struct{}, 10)

	for _, obj := range output.Buckets {
		blk <- struct{}{}
		go func(obj *s3.Bucket) {
			bucket := &Bucket{Name: obj.Name}
			region, err := bucket.getBucketLocation(c)
			if err != nil {
				panic(err)
			}

			// Ignore buckets in different region now
			if region != nil {
				ch <- &chanItem{}
				return
			}

			if err := bucket.GetBucketDetails(c); err != nil {
				ch <- &chanItem{err: err}
				return
			}

			ch <- &chanItem{obj: bucket, err: err}
			<-blk

		}(obj)

	}

	for range output.Buckets {
		receiver := <-ch
		if receiver.err != nil {
			return nil, err
		}

		if receiver.obj == nil {
			continue
		}

		res = append(res, receiver.obj.(*Bucket))
	}

	return &res, nil
}

func (b *Buckets) WriteHCL(w io.Writer) error {
	funcMap := template.FuncMap{
		"joinstring":       joinStringSlice,
		"StringValueSlice": aws.StringValueSlice,
		"replace":          strings.Replace,
		"prettyJSON":       prettyJSON,
	}

	tmpl := `
  {{- if .}}
    {{- range .}}
    resource "aws_s3_bucket" "{{ replace .Name "." "_" -1 }}" {
      bucket = "{{ .Name }}"

      {{- if .Logging}}
      logging {
        {{- if .Logging.TargetBucket}}
        target_bucket = "{{ .Logging.TargetBucket }}"
        {{- end}}
        {{- if .Logging.TargetPrefix}}
        target_prefix  = "{{ .Logging.TargetPrefix}}"
        {{- end}}
      }
      {{- end}}

      {{- if .Policy}}
      policy = <<POLICY
      {{ prettyJSON .Policy }}
POLICY
      {{- end}}

      {{- if .Versioning}}
      versioning {
        {{- if .Versioning.Enabled}}
        enabled = {{.Versioning.Enabled}}
        {{- end}}
        {{- if .Versioning.MFADelete}}
        mfa_delete = {{ .Versioning.MFADelete }}
        {{- end}}
      }
      {{- end}}

      {{- if .ServerSideEncryptionConfiguration }}
      server_side_encryption_configuration {
        {{- if .ServerSideEncryptionConfiguration.Rules}}
        {{- range .ServerSideEncryptionConfiguration.Rules}}
        rule {
          {{- if .ApplyServerSideEncryptionByDefault }}
          apply_server_side_encryption_by_default  {
            {{- if .ApplyServerSideEncryptionByDefault.KMSMasterKeyID}}
            kms_master_key_id = "{{.KMSMasterKeyID.KMSMasterKeyID }}"
            {{- end}}
            {{- if .ApplyServerSideEncryptionByDefault.SSEAlgorithm}}
            sse_algorithm = "{{ .ApplyServerSideEncryptionByDefault.SSEAlgorithm }}"
            {{- end}}
          }
          {{- end}}
        }
        {{- end}}
        {{- end}}
      }
      {{- end}}

      {{- if .ReplicationConfiguration}}
      replication_configuration {
        {{- if .ReplicationConfiguration.Role}}
        role = "{{ .ReplicationConfiguration.Role }}"
        {{- end}}
        {{- if .ReplicationConfiguration.Rules}}
        rules {
          {{- range .ReplicationConfiguration.Rules}}
          {{- if .ID}}
          id = "{{ .ID }}"
          {{- end}}

          {{- if .SourceSelectionCriteria}}
          source_selection_criteria {
            {{- if .SourceSelectionCriteria.SseKmsEncryptedObjects }}
            sse_kms_encrypted_objects {
              enabled = {{ .SourceSelectionCriteria.SseKmsEncryptedObjects.Status }}
            }
            {{- end }}
          }
          {{- end}}
          {{- end}}
          {{- if .LifecycleRules}}
          {{- range .LifecycleRules}}
          lifecycle_rule {
            id = "{{ .ID}}"
            prefix = "{{ .Prefix }}"
            enabled = {{ .Enable }}
            {{- if .NoncurrentVersionTransitions }}
            {{- range .}}
            noncurrent_version_transition  {
              {{- if .StorageClass}}
              storage_class = "{{ .StorageClass }}"
              {{- end}}
              {{- if .NoncurrentDays }}
              days = {{ .NoncurrentDays }}
              {{- end}}
            }
            {{- end}}
            {{- end }}

            {{- if .NoncurrentVersionExpiration }}
            {{- range .NoncurrentVersionExpiration}}
            noncurrent_version_expiration {
              days = {{ .NoncurrentDays }}
            }
            {{- end}}
            {{- end}}

            {{- if .Transition }}
            {{- range .Transition}}
            transition {
              {{- if .StorageClass}}
              storage_class = "{{ .StorageClass}}"
              {{- end}}
              {{- if .Days}}
              days = {{ .Days}}
              {{- end}}
              {{- if .Date}}
              date = "{{ .Date }}"
              {{- end}}
            }
            {{- end}}
            {{- end }}
          }
          {{- end}}
          {{- end }}

          {{- if .Prefix }}
          prefix = "{{ .Prefix}}"
          {{- end}}
          {{- if .Status}}
          status = "{{ .Status }}"
          {{- end}}
          {{- if .Destination }}
          {{- if .Destination.Bucket }}
          bucket = "{{ .Destination.Bucket }}"
          {{- end}}
          {{- if .Destination.StorageClass}}
          storage_class = "{{ .Destination.StorageClass}}"
          {{- end}}
          {{- if .Destination.EncryptionConfiguration}}
          replica_kms_key_id  = {{ .Destination.EncryptionConfiguration.ReplicaKmsKeyID}}
          {{- end }}
          {{- if .Destination.AccessControlTranslation}}
          access_control_translation = "{{ .Destination.AccessControlTranslation.Owner}}"
          {{- end}}
          {{- if .Destination.Account}}
          account_id = "{{ .Destination.Account }}"
          {{- end}}
          {{- end}}
        }
        {{- end}}
      }
      {{- end}}

      {{- if .CORSRules}}
       {{- range .CORSRules}}
      cors_rule {
        {{- if .AllowedHeaders }}
        {{- $allowheaders := StringValueSlice .AllowedHeaders}}
        allowed_headers = [{{ $allowheaders  | joinstring "," }}]
        {{- end }}
        {{- if .AllowedMethods}}
        {{- $allowmethods := StringValueSlice .AllowedMethods}}
        allowed_methods = [{{ $allowmethods | joinstring  "," }}]
        {{- end}}
        {{- if .AllowedOrigins}}
        {{- $allowOrigins := StringValueSlice .AllowedOrigins}}
        allowed_origins = [{{ $allowOrigins | joinstring ","}}]
        {{- end}}
        {{- if .ExposeHeaders}}
        {{- $exposeHeaders := StringValueSlice .ExposeHeaders}}
        expose_headers = [{{ $exposeHeaders  | joinstring ","}}]
        {{- end}}
        {{- if .MaxAgeSeconds}}
        max_age_seconds = {{ .MaxAgeSeconds}}
        {{- end}}
      }
       {{- end}}
      {{- end}}
    }
    {{- end }}
  {{- end }}
  `

	return renderHCL(w, tmpl, funcMap, b)
}
