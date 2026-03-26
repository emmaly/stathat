package stathat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPostCount(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithEZKey("test@example.com"), WithPostURL(srv.URL))
	err := c.PostCount(context.Background(), "page views", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vals, _ := parseFormBody(gotBody)
	assertFormValue(t, vals, "ezkey", "test@example.com")
	assertFormValue(t, vals, "stat", "page views")
	assertFormValue(t, vals, "count", "5")
}

func TestPostValue(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithEZKey("test@example.com"), WithPostURL(srv.URL))
	err := c.PostValue(context.Background(), "load average", 0.75)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vals, _ := parseFormBody(gotBody)
	assertFormValue(t, vals, "value", "0.75")
}

func TestPostCountAt(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithEZKey("test@example.com"), WithPostURL(srv.URL))
	ts := time.Unix(1700000000, 0)
	err := c.PostCountAt(context.Background(), "events", 1, ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vals, _ := parseFormBody(gotBody)
	assertFormValue(t, vals, "t", "1700000000")
}

func TestPostBatch(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithEZKey("test@example.com"), WithPostURL(srv.URL))
	err := c.PostBatch(context.Background(),
		CountReport("hits", 10),
		ValueReport("latency", 42.5),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req ezPostRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.EZKey != "test@example.com" {
		t.Errorf("ezkey: got %q, want %q", req.EZKey, "test@example.com")
	}
	if len(req.Data) != 2 {
		t.Fatalf("data items: got %d, want 2", len(req.Data))
	}
	if req.Data[0].Stat != "hits" || req.Data[0].Count == nil || *req.Data[0].Count != 10 {
		t.Errorf("first item: %+v", req.Data[0])
	}
	if req.Data[1].Stat != "latency" || req.Data[1].Value == nil || *req.Data[1].Value != 42.5 {
		t.Errorf("second item: %+v", req.Data[1])
	}
}

func TestPostBatchEmpty(t *testing.T) {
	c := New(WithEZKey("test@example.com"))
	err := c.PostBatch(context.Background())
	if !errors.Is(err, ErrEmptyBatch) {
		t.Errorf("expected ErrEmptyBatch, got %v", err)
	}
}

func TestPostNoEZKey(t *testing.T) {
	c := New()
	err := c.PostCount(context.Background(), "x", 1)
	if !errors.Is(err, ErrNoEZKey) {
		t.Errorf("expected ErrNoEZKey, got %v", err)
	}
}

func TestPostClassicCount(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/c" {
			t.Errorf("path: got %q, want /c", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithUserKey("user123"), WithPostURL(srv.URL))
	err := c.PostClassicCount(context.Background(), "stat456", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vals, _ := parseFormBody(gotBody)
	assertFormValue(t, vals, "key", "stat456")
	assertFormValue(t, vals, "ukey", "user123")
	assertFormValue(t, vals, "count", "3")
}

func TestPostClassicNoUserKey(t *testing.T) {
	c := New()
	err := c.PostClassicCount(context.Background(), "stat456", 1)
	if !errors.Is(err, ErrNoUserKey) {
		t.Errorf("expected ErrNoUserKey, got %v", err)
	}
}

func TestPostAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"msg":"invalid stat name"}`))
	}))
	defer srv.Close()

	c := New(WithEZKey("test@example.com"), WithPostURL(srv.URL))
	err := c.PostCount(context.Background(), "", 1)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("status: got %d, want 400", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid stat name" {
		t.Errorf("message: got %q, want %q", apiErr.Message, "invalid stat name")
	}
}

// parseFormBody parses a URL-encoded form body.
func parseFormBody(body string) (map[string]string, error) {
	result := make(map[string]string)
	if body == "" {
		return result, nil
	}
	for _, pair := range splitFormBody(body) {
		k, v, _ := splitFormPair(pair)
		result[k] = v
	}
	return result, nil
}

func splitFormBody(body string) []string {
	var parts []string
	for body != "" {
		idx := indexOf(body, '&')
		if idx < 0 {
			parts = append(parts, body)
			break
		}
		parts = append(parts, body[:idx])
		body = body[idx+1:]
	}
	return parts
}

func splitFormPair(pair string) (string, string, bool) {
	idx := indexOf(pair, '=')
	if idx < 0 {
		return pair, "", false
	}
	k, _ := unescape(pair[:idx])
	v, _ := unescape(pair[idx+1:])
	return k, v, true
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func unescape(s string) (string, error) {
	// Simple URL unescape for test purposes
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '+' {
			result = append(result, ' ')
		} else if s[i] == '%' && i+2 < len(s) {
			hi := unhex(s[i+1])
			lo := unhex(s[i+2])
			result = append(result, hi<<4|lo)
			i += 2
		} else {
			result = append(result, s[i])
		}
	}
	return string(result), nil
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func assertFormValue(t *testing.T, vals map[string]string, key, want string) {
	t.Helper()
	got, ok := vals[key]
	if !ok {
		t.Errorf("missing form key %q", key)
		return
	}
	if got != want {
		t.Errorf("form key %q: got %q, want %q", key, got, want)
	}
}
