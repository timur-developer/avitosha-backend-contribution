package usecase

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPhotoUploadServiceCreatesRestrictedSignedForm(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	objectID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	now := time.Date(2026, time.August, 12, 10, 30, 0, 0, time.UTC)
	service, err := NewPhotoUploadService(PhotoUploadConfig{
		Endpoint:        "/storage/",
		Region:          "us-east-1",
		Bucket:          "photos-test",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		PublicBaseURL:   "https://cdn.example.test/photos/",
		TTL:             10 * time.Minute,
		MaxFileSize:     10 * 1024 * 1024,
	}, func() uuid.UUID { return objectID })
	if err != nil {
		t.Fatalf("NewPhotoUploadService() error = %v", err)
	}

	form, err := service.CreateUpload(CreatePhotoUploadParams{
		UserID: userID, FileName: "sofa.jpg", ContentType: "image/jpeg", Size: 1024, Now: now,
	})
	if err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}

	wantKey := "listing-photos/11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222.jpg"
	if form.URL != "/storage/photos-test" || form.ObjectKey != wantKey {
		t.Fatalf("upload target = %q, key = %q", form.URL, form.ObjectKey)
	}
	if form.PublicURL != "https://cdn.example.test/photos/"+wantKey {
		t.Fatalf("PublicURL = %q", form.PublicURL)
	}
	if form.Fields["Content-Type"] != "image/jpeg" || form.Fields["key"] != wantKey {
		t.Fatalf("signed fields = %#v", form.Fields)
	}
	if _, exists := form.Fields["acl"]; exists {
		t.Fatal("object ACL must be controlled by the MinIO bucket policy")
	}
	if form.Fields["x-amz-signature"] == "" || strings.Contains(strings.Join(mapValues(form.Fields), " "), "test-secret-key") {
		t.Fatal("signature is missing or the secret leaked into the response")
	}

	policyJSON, err := base64.StdEncoding.DecodeString(form.Fields["policy"])
	if err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	var policy struct {
		Expiration string `json:"expiration"`
		Conditions []any  `json:"conditions"`
	}
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		t.Fatalf("decode policy JSON: %v", err)
	}
	if policy.Expiration != "2026-08-12T10:40:00Z" || !strings.Contains(string(policyJSON), `"content-length-range",1,10485760`) {
		t.Fatalf("policy = %s", policyJSON)
	}
}

func TestPhotoUploadServiceAgainstMinIO(t *testing.T) {
	endpoint := strings.TrimRight(os.Getenv("MINIO_TEST_ENDPOINT"), "/")
	if endpoint == "" {
		t.Skip("MINIO_TEST_ENDPOINT is not set")
	}

	service, err := NewPhotoUploadService(PhotoUploadConfig{
		Endpoint: endpoint, Region: "us-east-1", Bucket: "avitosha-photos",
		AccessKeyID: "avitosha-app", SecretAccessKey: "avitosha-app-secret-key",
		TTL: time.Minute, MaxFileSize: 1024,
	}, uuid.New)
	if err != nil {
		t.Fatalf("NewPhotoUploadService() error = %v", err)
	}
	payload := []byte{0xff, 0xd8, 0xff, 0xd9}
	form, err := service.CreateUpload(CreatePhotoUploadParams{
		UserID: uuid.New(), FileName: "integration.jpg", ContentType: "image/jpeg",
		Size: int64(len(payload)), Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	for key, value := range form.Fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %q: %v", key, err)
		}
	}
	filePart, err := writer.CreateFormFile("file", "integration.jpg")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := filePart.Write(payload); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, form.URL, &requestBody)
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("upload to MinIO: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("upload status = %d, body = %s", response.StatusCode, body)
	}

	getResponse, err := http.Get(form.PublicURL)
	if err != nil {
		t.Fatalf("read public object: %v", err)
	}
	defer getResponse.Body.Close()
	got, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("read object body: %v", err)
	}
	if getResponse.StatusCode != http.StatusOK || !bytes.Equal(got, payload) {
		t.Fatalf("public object status = %d, body = %v", getResponse.StatusCode, got)
	}
}

func TestPhotoUploadServiceRejectsInvalidFiles(t *testing.T) {
	t.Parallel()

	service, err := NewPhotoUploadService(PhotoUploadConfig{
		Endpoint: "/storage", Region: "us-east-1", Bucket: "photos-test",
		AccessKeyID: "access", SecretAccessKey: "secret", TTL: time.Minute, MaxFileSize: 100,
	}, uuid.New)
	if err != nil {
		t.Fatalf("NewPhotoUploadService() error = %v", err)
	}

	for _, params := range []CreatePhotoUploadParams{
		{UserID: uuid.New(), FileName: "photo.gif", ContentType: "image/gif", Size: 10},
		{UserID: uuid.New(), FileName: "photo.png", ContentType: "image/png", Size: 101},
		{UserID: uuid.Nil, FileName: "photo.webp", ContentType: "image/webp", Size: 10},
		{UserID: uuid.New(), FileName: "", ContentType: "image/jpeg", Size: 10},
	} {
		if _, err := service.CreateUpload(params); !errors.Is(err, ErrPhotoUploadInvalid) {
			t.Fatalf("CreateUpload(%+v) error = %v, want ErrPhotoUploadInvalid", params, err)
		}
	}
}

func mapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
