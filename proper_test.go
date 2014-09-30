package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/samalba/dockerclient"
)

const (
	dockerApiVersion          = "/v1.10"
	expectedDeletedContainers = 3
	expectedDeletedImages     = 4
)

var (
	client *dockerclient.DockerClient
	docker *httptest.Server

	serverT *testing.T

	matchContainersUrl = regexp.MustCompile("^" + dockerApiVersion + "/containers/([^/]+)")
	matchImagesUrl     = regexp.MustCompile("^" + dockerApiVersion + "/images/([^/]+)")
	deletedContainers  int
	deletedImages      int
)

func fixture(w http.ResponseWriter, file string) {
	fh, err := os.Open("fixtures/" + file)
	if err != nil {
		serverT.Fatal(err)
	}
	defer fh.Close()
	if _, err := io.Copy(w, fh); err != nil {
		serverT.Fatal(err)
	}
}

func handleDockerApi(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, dockerApiVersion+"/containers/json"):
		fixture(w, "containers.json")
		return
	case strings.HasPrefix(r.URL.Path, dockerApiVersion+"/containers/"):
		matches := matchContainersUrl.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			serverT.Fatal("ID not found in url ", r.URL.Path)
		}
		switch r.Method {
		case "GET":
			fixture(w, "containers/"+matches[1]+".json")
		case "DELETE":
			deletedContainers++
			fmt.Fprintf(w, "")
		}
		return
	case strings.HasPrefix(r.URL.Path, dockerApiVersion+"/images/json"):
		fixture(w, "images.json")
		return
	case strings.HasPrefix(r.URL.Path, dockerApiVersion+"/images"):
		matches := matchImagesUrl.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			serverT.Fatal("ID not found in url ", r.URL.Path)
		}
		deletedImages++
		return
	}
	serverT.Fatalf("Unexpected request: %s", r.URL.Path)
	return
}

func TestGetExpiredContainers(t *testing.T) {
	serverT = t
	docker = httptest.NewServer(http.HandlerFunc(handleDockerApi))
	defer docker.Close()

	client, err := dockerclient.NewDockerClient(docker.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := cleanup(client); err != nil {
		t.Fatal(err)
	}

	if deletedContainers != expectedDeletedContainers {
		t.Fatalf("Expected to delete %d container but deleted %d", expectedDeletedContainers, deletedContainers)
	}
	t.Log(deletedImages)
}