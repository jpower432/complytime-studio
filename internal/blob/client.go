// SPDX-License-Identifier: Apache-2.0

package blob

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// BlobStore uploads arbitrary bytes to object storage and returns a canonical ref.
type BlobStore interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64) (ref string, err error)
}

// MinioBlobStore implements BlobStore using the MinIO client (S3-compatible API).
type MinioBlobStore struct {
	cli    *minio.Client
	bucket string
}

// NewMinioBlobStore connects and ensures the bucket exists.
func NewMinioBlobStore(ctx context.Context, cfg Config) (*MinioBlobStore, error) {
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := cli.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}
	return &MinioBlobStore{cli: cli, bucket: cfg.Bucket}, nil
}

// Upload streams an object into the configured bucket and returns s3://bucket/key.
func (m *MinioBlobStore) Upload(ctx context.Context, key string, reader io.Reader, size int64) (string, error) {
	if key == "" {
		return "", fmt.Errorf("object key required")
	}
	_, err := m.cli.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("s3://%s/%s", m.bucket, key), nil
}

// EvidenceObjectKey builds a stable, non-colliding key for an evidence attachment.
func EvidenceObjectKey(originalFilename string) string {
	base := path.Base(strings.TrimSpace(originalFilename))
	if base == "" || base == "." {
		base = "attachment"
	}
	base = sanitizeFilename(base)
	return fmt.Sprintf("evidence/%s/%s", uuid.New().String(), base)
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := b.String()
	if s == "" || s == "." {
		return "attachment"
	}
	return s
}
