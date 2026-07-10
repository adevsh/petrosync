package service

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectStorage interface {
	PutObject(ctx context.Context, key string, body io.Reader, contentType string, sizeBytes int64) error
	PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type GarageStorage struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
}

func NewGarageStorage(ctx context.Context, endpoint, accessKey, secretKey, bucket string) (*GarageStorage, error) {
	region := "garage"
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     "garage",
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &GarageStorage{
		bucket:    bucket,
		client:    client,
		presigner: s3.NewPresignClient(client),
	}, nil
}

func (s *GarageStorage) PutObject(ctx context.Context, key string, body io.Reader, contentType string, sizeBytes int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(sizeBytes),
	})
	return err
}

func (s *GarageStorage) PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	out, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(o *s3.PresignOptions) {
		o.Expires = ttl
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}
