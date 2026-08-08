// Package blobstore abstracts where uploaded file bytes live. Historically
// every attachment and avatar sat in Postgres as BYTEA, which makes the
// database (and every dump) grow without bound and turns a full disk into a
// total outage. An S3-compatible object store keeps the bytes out of the
// database; Postgres keeps only metadata and the object key.
//
// The bytes are always served THROUGH the backend, never by handing clients a
// public object URL: access to a file is governed by chat membership and
// clearance, and only the backend can evaluate that.
package blobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotFound is returned when a key is absent from the store.
var ErrNotFound = errors.New("blobstore: object not found")

// Store is the persistence port for file bytes.
type Store interface {
	// Put stores data under key, overwriting any previous object.
	Put(ctx context.Context, key string, data []byte, contentType string) error
	// Get returns the object's bytes, or ErrNotFound.
	Get(ctx context.Context, key string) ([]byte, error)
	// Delete removes the object; deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
}

// Config describes an S3-compatible endpoint (MinIO self-hosted, AWS S3,
// Backblaze B2, Cloudflare R2 — all speak the same API).
type Config struct {
	Endpoint  string // host:port, or an https:// URL
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// Enabled reports whether object storage is configured. With it empty the
// caller keeps using the database-backed path.
func (c Config) Enabled() bool {
	return c.Endpoint != "" && c.Bucket != ""
}

type s3Store struct {
	client *minio.Client
	bucket string
}

// NewS3 connects to the endpoint and makes sure the bucket exists, so a
// misconfigured store fails at startup rather than on the first upload.
func NewS3(ctx context.Context, cfg Config) (Store, error) {
	endpoint, secure := normalizeEndpoint(cfg.Endpoint, cfg.UseSSL)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("blobstore: client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("blobstore: bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("blobstore: create bucket %q: %w", cfg.Bucket, err)
		}
	}

	return &s3Store{client: client, bucket: cfg.Bucket}, nil
}

// normalizeEndpoint accepts both "host:port" and a full URL, since operators
// paste either; the scheme (if any) decides TLS.
func normalizeEndpoint(raw string, useSSL bool) (string, bool) {
	if u, err := url.Parse(raw); err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") {
		return u.Host, u.Scheme == "https"
	}
	return raw, useSSL
}

func (s *s3Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("blobstore: put %q: %w", key, err)
	}
	return nil
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("blobstore: get %q: %w", key, err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("blobstore: read %q: %w", key, err)
	}
	return data, nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
		return fmt.Errorf("blobstore: delete %q: %w", key, err)
	}
	return nil
}
