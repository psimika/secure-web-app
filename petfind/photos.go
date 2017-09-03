package petfind

import (
	"encoding/base32"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
)

type Photo struct {
	ID               int64
	Key              string
	OriginalFilename string
	ContentType      string
	URL              string
	Created          time.Time
}

type PhotoStore interface {
	Upload(r io.Reader, contentType string) (*Photo, error)
	ServePhoto(w io.Writer, photo *Photo) error
}

type LocalPhotoStore struct {
	photosPath string
}

func NewPhotoStore(uploadPath string) *LocalPhotoStore {
	return &LocalPhotoStore{photosPath: uploadPath}
}

func (s *LocalPhotoStore) ServePhoto(w io.Writer, photo *Photo) error {
	path := filepath.Join(s.photosPath, photo.Key)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return ErrNotFound
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening photo from disk")
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("error serving photo from disk")
	}
	return nil
}

func (s *LocalPhotoStore) Upload(r io.Reader, contentType string) (*Photo, error) {
	key := securecookie.GenerateRandomKey(32)
	if key == nil {
		return nil, fmt.Errorf("error generating random key for photo")
	}
	photoKey := strings.TrimRight(base32.StdEncoding.EncodeToString(key), "=")

	if err := mkDirAllIfNotExist(s.photosPath, 0700); err != nil {
		return nil, fmt.Errorf("error creating upload dir: %v", err)
	}

	f, err := os.Create(filepath.Join(s.photosPath, photoKey))
	if err != nil {
		return nil, fmt.Errorf("error creating file for upload: %v", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return nil, fmt.Errorf("error copying upload file: %v", err)
	}

	photo := &Photo{
		Key:         photoKey,
		ContentType: contentType,
	}
	return photo, nil
}

func mkDirAllIfNotExist(name string, perm os.FileMode) error {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(name, perm); err != nil {
				return fmt.Errorf("could not make dir %q: %v", name, err)
			}
			log.Printf("Created directory %s %v", name, perm)
		}
	}
	return nil
}
