package cloudinary

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

type Store struct {
	apiKey    string
	apiSecret string
	cloudName string
}

func NewPhotoStore(apiKey, apiSecret, cloudName string) *Store {
	return &Store{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		cloudName: cloudName,
	}
}

func (s *Store) ServePhoto(w io.Writer, photo *petfind.Photo) error {
	resp, err := http.Get(photo.URL)
	if err != nil {
		return fmt.Errorf("error fetching cloudinary image: %v", err)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("Error serving cloudinary image: %v", err)
	}
	return nil
}

func (s *Store) Upload(r io.Reader, contentType string) (*petfind.Photo, error) {
	// Prepare API parameters.
	v := url.Values{}
	v.Add("api_key", s.apiKey)
	v.Add("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	sig := generateSignature(v, s.apiSecret)
	v.Add("signature", sig)

	// Read file and generate data URI base64 format.
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if _, err := io.Copy(w, r); err != nil {
		return nil, fmt.Errorf("failed to read upload file: %v", err)
	}
	data := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, data)
	v.Add("file", dataURI)

	// Prepare Cloudinary API request.
	u := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", s.cloudName)
	req, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error preparing cloudinary request: %v", err)
	}

	// Do request.
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudinary request failed: %v", err)
	}
	defer resp.Body.Close()

	// Decode Cloudinary API response.
	upload := new(upload)
	if err := json.NewDecoder(resp.Body).Decode(upload); err != nil {
		return nil, fmt.Errorf("error decoding cloudinary response: %v", err)
	}

	photo := &petfind.Photo{
		ContentType: contentType,
		URL:         upload.SecureURL,
		Created:     upload.CreatedAt,
	}
	return photo, nil
}

type upload struct {
	PublicID     string    `json:"public_id"`
	Version      int       `json:"version"`
	Signature    string    `json:"signature"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Format       string    `json:"format"`
	ResourceType string    `json:"resource_type"`
	CreatedAt    time.Time `json:"created_at"`
	Tags         []string  `json:"tags"`
	Bytes        int       `json:"bytes"`
	Type         string    `json:"type"`
	Etag         string    `json:"etag"`
	URL          string    `json:"url"`
	SecureURL    string    `json:"secure_url"`
}

func generateSignature(values url.Values, secret string) string {
	sortedKeys := []string{}
	for k := range values {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	sigValues := make([]string, 0)
	for _, k := range sortedKeys {
		if k == "file" || k == "type" || k == "resource_type" || k == "api_key" {
			continue
		}
		if v, ok := values[k]; ok {
			sigValues = append(sigValues, fmt.Sprintf("%s=%s", k, v[0]))
		}
	}

	signature := strings.Join(sigValues, "&")
	hash := sha1.New()
	hash.Write([]byte(signature + secret))
	return hex.EncodeToString(hash.Sum(nil))
}
