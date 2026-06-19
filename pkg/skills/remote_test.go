package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

const remoteSkillBody = "---\nname: remote-skill\ndescription: A remote skill\n---\nremote body"

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestFetchRemoteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(remoteSkillBody))
	}))
	defer srv.Close()

	src := RemoteSource{Source: srv.URL + "/skill.md", SHA256: sha256Hex(remoteSkillBody)}
	opts := FetchOptions{CacheDir: t.TempDir(), AllowedHosts: []string{"127.0.0.1"}}

	skill, err := FetchRemote(context.Background(), src, opts)
	if err != nil {
		t.Fatalf("FetchRemote: %v", err)
	}
	if skill.Name != "remote-skill" || skill.Content != "remote body" {
		t.Errorf("unexpected skill: %+v", skill)
	}
}

func TestFetchRemoteChecksumMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(remoteSkillBody))
	}))
	defer srv.Close()

	src := RemoteSource{Source: srv.URL + "/skill.md", SHA256: sha256Hex("something else")}
	opts := FetchOptions{CacheDir: t.TempDir(), AllowedHosts: []string{"127.0.0.1"}}

	if _, err := FetchRemote(context.Background(), src, opts); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestFetchRemoteHostNotAllowed(t *testing.T) {
	src := RemoteSource{Source: "https://evil.example.com/skill.md"}
	// Default allowlist excludes evil.example.com.
	if _, err := FetchRemote(context.Background(), src, FetchOptions{CacheDir: t.TempDir()}); err == nil {
		t.Fatal("expected host-not-allowed error")
	}
}

func TestFetchRemoteUsesCache(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(remoteSkillBody))
	}))

	cache := t.TempDir()
	src := RemoteSource{Source: srv.URL + "/skill.md", SHA256: sha256Hex(remoteSkillBody)}
	opts := FetchOptions{CacheDir: cache, AllowedHosts: []string{"127.0.0.1"}}

	if _, err := FetchRemote(context.Background(), src, opts); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	srv.Close() // second fetch must come from cache

	skill, err := FetchRemote(context.Background(), src, opts)
	if err != nil {
		t.Fatalf("cached fetch: %v", err)
	}
	if skill.Content != "remote body" {
		t.Errorf("unexpected cached content: %q", skill.Content)
	}
	if hits != 1 {
		t.Errorf("expected exactly 1 network hit, got %d", hits)
	}
}

func TestResolveURLGitHub(t *testing.T) {
	got, err := resolveURL(RemoteSource{Source: "github.com/acme/repo/skills/refunds.md", Ref: "abc123"})
	if err != nil {
		t.Fatalf("resolveURL: %v", err)
	}
	want := "https://raw.githubusercontent.com/acme/repo/abc123/skills/refunds.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveURLGitHubRequiresRef(t *testing.T) {
	if _, err := resolveURL(RemoteSource{Source: "github.com/acme/repo/skill.md"}); err == nil {
		t.Error("expected error when ref is missing")
	}
}

func TestResolveURLUnsupported(t *testing.T) {
	if _, err := resolveURL(RemoteSource{Source: "ftp://example.com/x"}); err == nil {
		t.Error("expected unsupported-source error")
	}
}
