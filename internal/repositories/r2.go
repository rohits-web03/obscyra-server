package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	R2Client     *s3.Client
	R2BucketName string
	R2Endpoint   string
)

// InitR2 initializes the R2 client using static credentials and custom endpoint.
func InitR2(accessKey, secretKey, accountID, bucketName, region string) error {
	R2BucketName = bucketName
	R2Endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		Region:      region,
	}

	R2Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(R2Endpoint)
		o.UsePathStyle = true
	})

	log.Println("Successfully initialized R2 client")

	return nil
}

// GeneratePresignedPutURL creates a presigned URL for uploading a file to R2.
func GeneratePresignedPutURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presigner := s3.NewPresignClient(R2Client)
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(R2BucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// GeneratePresignedGetURL creates a presigned URL for downloading a file from R2.
func GeneratePresignedGetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presigner := s3.NewPresignClient(R2Client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(R2BucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// VerifyObjectExists checks if a given object key exists in the R2 bucket.
// Returns true if the object exists, false if not, and an error if something went wrong.
func VerifyObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := R2Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(R2BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *s3types.NotFound
		if ok := errors.As(err, &nsk); ok {
			// Object not found
			return false, nil
		}
		// Other error (e.g. auth, network)
		return false, err
	}
	return true, nil
}
