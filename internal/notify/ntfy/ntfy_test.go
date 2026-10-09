package ntfy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSend(t *testing.T) {
	var gotPath, gotTitle, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTitle = r.Header.Get("Title")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer srv.Close()

	c := &Client{Server: srv.URL + "/", Topic: "my-topic", Token: "tk"}
	if err := c.Send(context.Background(), "my-app", "Claude finished"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/my-topic" || gotTitle != "my-app" || gotAuth != "Bearer tk" || gotBody != "Claude finished" {
		t.Errorf("path=%q title=%q auth=%q body=%q", gotPath, gotTitle, gotAuth, gotBody)
	}
}

func TestSendNoTokenNoAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["Authorization"]; ok {
			t.Error("Authorization header sent without a token")
		}
	}))
	defer srv.Close()
	c := &Client{Server: srv.URL, Topic: "t"}
	if err := c.Send(context.Background(), "a", "b"); err != nil {
		t.Fatal(err)
	}
}

func TestSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden topic", http.StatusForbidden)
	}))
	defer srv.Close()
	c := &Client{Server: srv.URL, Topic: "t"}
	err := c.Send(context.Background(), "a", "b")
	if err == nil || !strings.Contains(err.Error(), "forbidden topic") {
		t.Fatalf("err = %v", err)
	}
}
