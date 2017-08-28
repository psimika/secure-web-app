package cloudinary

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	cloudinary "github.com/b4t3ou/cloudinary-go"
	"github.com/psimika/secure-web-app/petfind"
)

type Store struct {
	id        string
	secret    string
	cloudName string
	client    *cloudinary.Cloudinary
}

func NewPhotoStore(id, secret, cloudName string) *Store {
	return &Store{client: cloudinary.Create(id, secret, cloudName)}
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
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if _, err := io.Copy(w, r); err != nil {
		return nil, fmt.Errorf("failed to read upload file: %v", err)
	}

	data := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, data)

	options := map[string]string{}
	u, err := s.client.Upload(dataURI, options)
	if err != nil {
		return nil, err
	}
	if u.Error.Message != "" {
		return nil, fmt.Errorf("%s", u.Error.Message)
	}

	photo := &petfind.Photo{
		ContentType: contentType,
		URL:         u.SecureUrl,
	}
	return photo, nil
}
