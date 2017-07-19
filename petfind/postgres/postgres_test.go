// +build db

package postgres_test

import (
	"flag"
	"fmt"
	"testing"

	"github.com/psimika/secure-web-app/petfind"
	"github.com/psimika/secure-web-app/petfind/postgres"
)

var (
	user   = flag.String("user", "psimika", "username to use to run tests on Postgres")
	pass   = flag.String("pass", "", "password to use to run tests on Postgres")
	host   = flag.String("host", "localhost", "host for connecting to Postgres and run the tests")
	port   = flag.String("port", "5432", "port for connecting to Postgres and run the tests")
	dbname = flag.String("dbname", "petfind_test", "test database, tables will be created and dropped on each test")
)

func init() {
	flag.Parse()
}

// To include the test coverage of the db tests:
//
// go test -tags=db -pass '' -coverprofile=cover.out -covermode=count
//
// go tool cover -html=cover.out
//
// (Pike 2013) https://blog.golang.org/cover

func setup(t *testing.T) petfind.Store {
	if *pass == "" {
		t.Errorf("No password provided for user %q to connect to Postgres and run the tests.", *user)
		t.Errorf("These tests need a Postgres account %q that has access to test database %q.", *user, *dbname)
		t.Fatal("Use: go test -tags=db -pass '<db password>'")
	}
	datasource := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s", *user, *pass, *host, *port, *dbname)
	s, err := postgres.NewStore(datasource)
	if err != nil {
		t.Fatalf("NewStore failed for datasource %q: %v", datasource, err)
	}
	return s
}

func teardown(t *testing.T, s petfind.Store) {
	if err := s.DropSchema(); err != nil {
		t.Error(err)
	}
}
