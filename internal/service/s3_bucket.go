package services

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	BucketName string
	Client     *s3.Client
}

// NewS3Service initializes the S3 service
func NewS3Service() (*S3Service, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),                             // Cambia según la región de tu bucket
		config.WithCredentialsProvider(aws.AnonymousCredentials{}), // Configura el acceso anónimo
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name is not set in environment variables")
	}

	client := s3.NewFromConfig(cfg)

	return &S3Service{
		BucketName: bucketName,
		Client:     client,
	}, nil
}

// UploadFile uploads a file to the S3 bucket and returns the public URL
func UploadFile(s *S3Service, file multipart.File, fileHeader *multipart.FileHeader, downloadable bool) (string, error) {
	defer file.Close()

	// Read file content
	buffer := bytes.NewBuffer(nil)
	if _, err := buffer.ReadFrom(file); err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Generate a unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), fileHeader.Filename)

	putObjectInput := &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(filename),
		Body:   bytes.NewReader(buffer.Bytes()),
	}

	if !downloadable {
		putObjectInput.ContentType = aws.String(fileHeader.Header.Get("Content-Type"))
	}

	// Upload the file
	_, err := s.Client.PutObject(context.TODO(), putObjectInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	// Generate the public URL
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.BucketName, filename)
	return url, nil
}
