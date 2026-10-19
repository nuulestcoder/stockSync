package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Config struct {
	Mode      string
	UploadDir string
	PublicURL string
	R2        R2Config
}

type R2Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	PublicURL string
}

type File struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type Store interface {
	Upload(ctx context.Context, companyID bson.ObjectID, name, contentType string, size int64, reader io.Reader) (File, error)
}

func New(ctx context.Context, cfg Config) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "local":
		if cfg.UploadDir == "" {
			cfg.UploadDir = "./uploads"
		}
		if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
			return nil, fmt.Errorf("storage: create upload directory: %w", err)
		}
		return &localStore{directory: cfg.UploadDir, publicURL: strings.TrimRight(cfg.PublicURL, "/")}, nil
	case "r2":
		if cfg.R2.Endpoint == "" || cfg.R2.Bucket == "" || cfg.R2.AccessKey == "" || cfg.R2.SecretKey == "" {
			return nil, fmt.Errorf("storage: R2 endpoint, bucket, access key, and secret key are required")
		}
		awsConfig, err := config.LoadDefaultConfig(ctx,
			config.WithRegion("auto"),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2.AccessKey, cfg.R2.SecretKey, "")),
		)
		if err != nil {
			return nil, fmt.Errorf("storage: load R2 configuration: %w", err)
		}
		return &r2Store{
			client:    s3.NewFromConfig(awsConfig, func(options *s3.Options) { options.BaseEndpoint = aws.String(cfg.R2.Endpoint) }),
			bucket:    cfg.R2.Bucket,
			publicURL: strings.TrimRight(cfg.R2.PublicURL, "/"),
		}, nil
	default:
		return nil, fmt.Errorf("storage: unsupported mode %q", cfg.Mode)
	}
}

type localStore struct {
	directory string
	publicURL string
}

func (s *localStore) Upload(_ context.Context, companyID bson.ObjectID, name, contentType string, size int64, reader io.Reader) (File, error) {
	key := filepath.Join(companyID.Hex(), bson.NewObjectID().Hex()+filepath.Ext(name))
	path := filepath.Join(s.directory, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return File{}, fmt.Errorf("storage: create tenant upload directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return File{}, fmt.Errorf("storage: create uploaded file: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return File{}, fmt.Errorf("storage: write uploaded file: %w", err)
	}
	return File{Key: filepath.ToSlash(key), URL: s.url(key), Name: filepath.Base(name), ContentType: contentType, Size: size}, nil
}

func (s *localStore) url(key string) string {
	if s.publicURL == "" {
		return "/uploads/" + filepath.ToSlash(key)
	}
	return s.publicURL + "/" + filepath.ToSlash(key)
}

type r2Store struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func (s *r2Store) Upload(ctx context.Context, companyID bson.ObjectID, name, contentType string, size int64, reader io.Reader) (File, error) {
	key := companyID.Hex() + "/" + bson.NewObjectID().Hex() + filepath.Ext(name)
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}); err != nil {
		return File{}, fmt.Errorf("storage: upload file to R2: %w", err)
	}
	url := key
	if s.publicURL != "" {
		url = s.publicURL + "/" + key
	}
	return File{Key: key, URL: url, Name: filepath.Base(name), ContentType: contentType, Size: size}, nil
}
