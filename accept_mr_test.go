package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/api/client-go"
)

func TestAcceptMr_Run(t *testing.T) {
	// Create a test server to mock GitLab API responses
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		regexpMrNotes := regexp.MustCompile("merge_requests/([0-9]+)/notes")
		regexpMrMerge := regexp.MustCompile("merge_requests/([0-9]+)/merge")
		regexpMr := regexp.MustCompile("merge_requests(,|$)")
		fmt.Println(r.URL.Path)
		if regexpMrNotes.MatchString(r.URL.Path) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"body": "test mr note"}`))
			if err != nil {
				t.Error(err)
			}
		}
		if regexpMrMerge.MatchString(r.URL.Path) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"state": "merged"}`))
			if err != nil {
				t.Error(err)
			}
		}
		if regexpMr.MatchString(r.URL.Path) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[{"title": "Test MR", "state": "opened"}]`))
			if err != nil {
				t.Error(err)
			}
		}
		if strings.Contains(r.URL.Path, "commits") {
			w.WriteHeader(http.StatusOK)
		}
		if strings.Contains(r.URL.Path, "statuses") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[{"status": "opened"}]`))
			if err != nil {
				t.Error(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Create a GitLab client pointing to the test server
	client, err := gitlab.NewClient("", gitlab.WithBaseURL(ts.URL))
	assert.NoError(t, err)

	// Create an instance of AcceptMr with the mocked client
	acceptMr := AcceptMr{
		Client:             client,
		OnBuildSucceed:     true,
		RemoveSourceBranch: true,
		PipelineName:       "test-pipeline",
		PipelineState:      "success",
		ProjectName:        "test-project",
		FailOnError:        true,
		Message:            "Merging MR",
	}

	// Run the method and check for errors
	err = acceptMr.Run()
	assert.NoError(t, err)
}
