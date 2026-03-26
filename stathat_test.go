package stathat

import (
	"log/slog"
	"net/http"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	c := New()
	if c.httpClient != http.DefaultClient {
		t.Error("expected default HTTP client")
	}
	if c.postURL != defaultPostURL {
		t.Errorf("postURL: got %q, want %q", c.postURL, defaultPostURL)
	}
	if c.exportURL != defaultExportURL {
		t.Errorf("exportURL: got %q, want %q", c.exportURL, defaultExportURL)
	}
}

func TestNewWithOptions(t *testing.T) {
	custom := &http.Client{}
	logger := slog.Default()

	c := New(
		WithHTTPClient(custom),
		WithEZKey("ez@test.com"),
		WithUserKey("ukey"),
		WithAccessToken("atok"),
		WithLogger(logger),
		WithPostURL("http://post.test/"),
		WithExportURL("http://export.test/"),
	)

	if c.httpClient != custom {
		t.Error("expected custom HTTP client")
	}
	if c.ezKey != "ez@test.com" {
		t.Errorf("ezKey: got %q", c.ezKey)
	}
	if c.userKey != "ukey" {
		t.Errorf("userKey: got %q", c.userKey)
	}
	if c.accessToken != "atok" {
		t.Errorf("accessToken: got %q", c.accessToken)
	}
	if c.postURL != "http://post.test" {
		t.Errorf("postURL: got %q (trailing slash should be trimmed)", c.postURL)
	}
	if c.exportURL != "http://export.test" {
		t.Errorf("exportURL: got %q (trailing slash should be trimmed)", c.exportURL)
	}
}

func TestExportPathRequiresToken(t *testing.T) {
	c := New()
	_, err := c.exportPath("/statlist")
	if err != ErrNoAccessToken {
		t.Errorf("expected ErrNoAccessToken, got %v", err)
	}
}

func TestExportPathFormat(t *testing.T) {
	c := New(WithAccessToken("mytoken"))
	path, err := c.exportPath("/statlist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://www.stathat.com/x/mytoken/statlist"
	if path != want {
		t.Errorf("got %q, want %q", path, want)
	}
}
