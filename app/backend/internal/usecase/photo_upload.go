package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

const photoCacheControl = "public,max-age=31536000,immutable"

var (
	ErrPhotoUploadInvalid   = errors.New("photo upload is invalid")
	ErrPhotoStorageDisabled = errors.New("photo storage is not configured")
)

type PhotoUploadConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	PublicBaseURL   string
	TTL             time.Duration
	MaxFileSize     int64
}

type CreatePhotoUploadParams struct {
	UserID      uuid.UUID
	FileName    string
	ContentType string
	Size        int64
	Now         time.Time
}

type PhotoUploadForm struct {
	URL         string            `json:"url"`
	Fields      map[string]string `json:"fields"`
	PublicURL   string            `json:"publicUrl"`
	ObjectKey   string            `json:"objectKey"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	MaxFileSize int64             `json:"maxFileSize"`
}

type PhotoUploadService struct {
	config      PhotoUploadConfig
	idGenerator func() uuid.UUID
}

func NewPhotoUploadService(config PhotoUploadConfig, idGenerator func() uuid.UUID) (*PhotoUploadService, error) {
	if strings.TrimSpace(config.Endpoint) == "" || strings.TrimSpace(config.Region) == "" ||
		strings.TrimSpace(config.Bucket) == "" || strings.TrimSpace(config.AccessKeyID) == "" ||
		strings.TrimSpace(config.SecretAccessKey) == "" || config.TTL <= 0 || config.MaxFileSize <= 0 {
		return nil, fmt.Errorf("photo upload config: %w", ErrPhotoStorageDisabled)
	}
	endpointValue := strings.TrimRight(config.Endpoint, "/")
	endpoint, err := url.Parse(endpointValue)
	validRelativeEndpoint := strings.HasPrefix(endpointValue, "/") && endpoint.Host == "" && endpoint.RawQuery == "" && endpoint.Fragment == ""
	validAbsoluteEndpoint := endpoint.Scheme != "" && endpoint.Host != ""
	if err != nil || (!validRelativeEndpoint && !validAbsoluteEndpoint) {
		return nil, fmt.Errorf("photo upload endpoint: %w", ErrPhotoStorageDisabled)
	}
	if idGenerator == nil {
		idGenerator = uuid.New
	}
	config.Endpoint = endpointValue
	config.PublicBaseURL = strings.TrimRight(config.PublicBaseURL, "/")
	return &PhotoUploadService{config: config, idGenerator: idGenerator}, nil
}

func (service *PhotoUploadService) CreateUpload(params CreatePhotoUploadParams) (PhotoUploadForm, error) {
	extension, ok := photoExtension(strings.TrimSpace(params.ContentType))
	fileName := strings.TrimSpace(params.FileName)
	if params.UserID == uuid.Nil || fileName == "" || len(fileName) > 255 || !ok || params.Size <= 0 || params.Size > service.config.MaxFileSize {
		return PhotoUploadForm{}, ErrPhotoUploadInvalid
	}
	now := params.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	expiresAt := now.Add(service.config.TTL)
	objectKey := path.Join("listing-photos", params.UserID.String(), service.idGenerator().String()+extension)
	date := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	credential := service.config.AccessKeyID + "/" + date + "/" + service.config.Region + "/s3/aws4_request"

	conditions := []any{
		map[string]string{"bucket": service.config.Bucket},
		map[string]string{"key": objectKey},
		map[string]string{"Content-Type": params.ContentType},
		map[string]string{"Cache-Control": photoCacheControl},
		map[string]string{"success_action_status": "204"},
		map[string]string{"x-amz-algorithm": "AWS4-HMAC-SHA256"},
		map[string]string{"x-amz-credential": credential},
		map[string]string{"x-amz-date": amzDate},
		[]any{"content-length-range", 1, service.config.MaxFileSize},
	}
	policyJSON, err := json.Marshal(map[string]any{
		"expiration": expiresAt.Format("2006-01-02T15:04:05Z"),
		"conditions": conditions,
	})
	if err != nil {
		return PhotoUploadForm{}, fmt.Errorf("encode upload policy: %w", err)
	}
	policy := base64.StdEncoding.EncodeToString(policyJSON)
	signature := signS3Policy(service.config.SecretAccessKey, date, service.config.Region, policy)
	actionURL := service.config.Endpoint + "/" + service.config.Bucket
	publicBaseURL := service.config.PublicBaseURL
	if publicBaseURL == "" {
		publicBaseURL = actionURL
	}

	return PhotoUploadForm{
		URL: actionURL, PublicURL: publicBaseURL + "/" + objectKey, ObjectKey: objectKey,
		ExpiresAt: expiresAt, MaxFileSize: service.config.MaxFileSize,
		Fields: map[string]string{
			"key": objectKey, "Content-Type": params.ContentType,
			"Cache-Control": photoCacheControl, "success_action_status": "204",
			"x-amz-algorithm": "AWS4-HMAC-SHA256", "x-amz-credential": credential,
			"x-amz-date": amzDate, "policy": policy, "x-amz-signature": signature,
		},
	}, nil
}

func photoExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func signS3Policy(secret, date, region, policy string) string {
	dateKey := hmacSHA256([]byte("AWS4"+secret), date)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, "s3")
	signingKey := hmacSHA256(serviceKey, "aws4_request")
	return hex.EncodeToString(hmacSHA256(signingKey, policy))
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
