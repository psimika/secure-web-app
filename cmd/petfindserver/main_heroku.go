// +build heroku

package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/boj/redistore"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"

	"github.com/psimika/secure-web-app/petfind"
	"github.com/psimika/secure-web-app/petfind/cloudinary"
	"github.com/psimika/secure-web-app/petfind/postgres"
	"github.com/psimika/secure-web-app/web"
)

// This is an implementation of make.go that is specific to Heroku as indicated
// by the build tag. The build tag means that on Heroku the main_heroku.go will
// be built and on any other case main.go will be built instead.
//
// In order to deploy to Heroku for the first time we need these steps:
//
//     heroku login
//
//     heroku create
//
//     heroku addons:create heroku-postgresql:hobby-dev
//
// After that and each time we make a change on master branch:
//
//     git push heroku master
//
// Or when working on a different branch:
//
//     git push heroku somebranch:master

func main() {
	// Heroku uses the environment variables DATABASE_URL and PORT so that the
	// app knows on which database to connect and on which port to listen on.
	// Heroku deploys the application under /app.
	var (
		databaseURL      = getenvString("", "DATABASE_URL")
		port             = getenvString("8080", "PORT")
		tmplPath         = getenvString("/app/web", "TMPL_PATH")
		photosPath       = getenvString("/app/photos", "PHOTOS_PATH")
		githubID         = getenvString("", "GITHUB_ID")
		githubSecret     = getenvString("", "GITHUB_SECRET")
		sessionTTL       = getenvInt(1200, "SESSION_TTL")
		sessionMaxTTL    = getenvInt(3600, "SESSION_MAX_TTL")
		redisURL         = getenvString("", "REDIS_URL")
		redisPass        = getenvString("", "REDIS_PASS")
		redisMaxIdle     = getenvInt(10, "REDIS_MAX_IDLE")
		hashKeyStr       = getenvString("", "HASH_KEY")
		blockKeyStr      = getenvString("", "BLOCK_KEY")
		csrfKeyStr       = getenvString("", "CSRF_KEY")
		cloudinaryKey    = getenvString("", "CLOUDINARY_KEY")
		cloudinarySecret = getenvString("", "CLOUDINARY_SECRET")
		cloudinaryName   = getenvString("petfind-photos", "CLOUDINARY_NAME")
	)
	hashKey := validHashKey(hashKeyStr)
	blockKey := validBlockKey(blockKeyStr)
	csrfKey := validCSRFKey(csrfKeyStr)

	if databaseURL == "" {
		log.Fatal("No database URL provided, exiting...")
	}

	store, err := postgres.NewStore(databaseURL)
	if err != nil {
		log.Println("NewStore failed:", err)
		return
	}

	sessionStore, err := newRediStoreURL(redisMaxIdle, redisURL, redisPass, hashKey, blockKey)
	if err != nil {
		log.Println("NewRediStore failed:", err)
		return
	}
	defer sessionStore.Close()
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   sessionTTL,
	}

	CSRF := csrf.Protect(csrfKey)

	var photos petfind.PhotoStore
	if cloudinaryKey != "" && cloudinarySecret != "" && cloudinaryName != "" {
		photos = cloudinary.NewPhotoStore(*cloudinaryKey, *cloudinarySecret, *cloudinaryName)
	} else {
		log.Println("Warning: Using local photo store. Photos will be deleted on app restart!")
		photos = petfind.NewPhotoStore(*photosPath)
	}

	handlers, err := web.NewServer(
		store,
		sessionStore,
		sessionTTL,
		sessionMaxTTL,
		CSRF,
		tmplPath,
		photoStore,
		githubID,
		githubSecret)
	if err != nil {
		log.Println("NewServer failed:", err)
		return
	}

	log.Fatal(http.ListenAndServe(":"+port, redirectHTTP(handlers)))
}

func getenvString(defaultValue, envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		return defaultValue
	}
	return value
}

func getenvInt(defaultValue int, envName string) int {
	value := os.Getenv(envName)
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

func redirectHTTP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Heroku's HTTP routing passes requests to our app and uses the
		// X-Forwarded-Proto header to carry the information about the
		// originating protocol of the HTTP request. Here we check that header
		// and if the original request was HTTP we perform a redirect to HTTPS
		// (Heroku Dev Center 2017):
		//
		// https://devcenter.heroku.com/articles/http-routing#heroku-headers
		if r.Header.Get("X-Forwarded-Proto") == "http" {
			w.Header().Set("Connection", "close")
			u := r.URL
			u.Host = r.Host
			u.Scheme = "https"
			http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func newRediStoreURL(size int, url, password string, keyPairs ...[]byte) (*redistore.RediStore, error) {
	return redistore.NewRediStoreWithPool(&redis.Pool{
		MaxIdle:     size,
		IdleTimeout: 240 * time.Second,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		Dial: func() (redis.Conn, error) {
			return dialURL(url, password)
		},
	}, keyPairs...)
}

func dialURL(url, password string) (redis.Conn, error) {
	c, err := redis.DialURL(url)
	if err != nil {
		return nil, err
	}
	if password != "" {
		if _, err = c.Do("AUTH", password); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, err
}
