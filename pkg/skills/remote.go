package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// RemoteSource declares a skill fetched from a remote location. Sources are
// declared by the developer (never supplied by the model) and are resolved at
// load time, so a malicious skill cannot be injected at runtime.
//
// Supported source forms:
//   - github.com/<org>/<repo>/<path...>  (requires Ref, a commit SHA)
//   - https://<host>/<path...>           (a direct URL; pin it yourself)
//
// Always set Ref to an immutable commit SHA for GitHub sources, and prefer
// setting SHA256 so the fetched bytes are integrity-checked.
//
// Security note: Ref is only required to be non-empty. Pointing it at a mutable
// branch or tag (rather than a commit SHA) defeats the immutability guarantee —
// the upstream content can change underneath you. Without SHA256, cached bytes
// are served without revalidation. Pin a SHA for anything you don't control.
type RemoteSource struct {
	Source string `yaml:"source"`
	Ref    string `yaml:"ref"`
	SHA256 string `yaml:"sha256"`
	Name   string `yaml:"name"`
}

// FetchOptions configures remote skill fetching.
type FetchOptions struct {
	// CacheDir is where fetched skills are cached. Empty uses the OS cache dir.
	CacheDir string
	// AllowedHosts restricts which hosts may be fetched. Empty uses
	// DefaultAllowedHosts.
	AllowedHosts []string
	// Client is the HTTP client to use. Empty uses a client with a 30s timeout.
	Client *http.Client
}

// DefaultAllowedHosts is the conservative default allowlist of skill hosts.
var DefaultAllowedHosts = []string{
	"raw.githubusercontent.com",
	"gist.githubusercontent.com",
}

// FetchRemote resolves, fetches (or reads from cache), integrity-checks, and
// parses a single remote skill.
func FetchRemote(ctx context.Context, src RemoteSource, opts FetchOptions) (Skill, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	rawURL, err := resolveURL(src)
	if err != nil {
		return Skill{}, err
	}
	if err := checkHost(rawURL, allowedHosts(opts)); err != nil {
		return Skill{}, err
	}

	cacheDir, err := resolveCacheDir(opts.CacheDir)
	if err != nil {
		return Skill{}, err
	}
	// Key the cache by the fully resolved URL so distinct sources/refs can never
	// collide on a single cache entry.
	cachePath := filepath.Join(cacheDir, cacheKey(rawURL)+".md")

	data, err := readCache(cachePath, src.SHA256)
	if err != nil {
		return Skill{}, err
	}
	if data == nil {
		data, err = download(ctx, rawURL, opts, allowedHosts(opts))
		if err != nil {
			return Skill{}, err
		}
		if err := verifyChecksum(data, src.SHA256); err != nil {
			return Skill{}, fmt.Errorf("skill %s: %w", src.Source, err)
		}
		if err := writeCache(cachePath, data); err != nil {
			return Skill{}, err
		}
	}

	defaultName := src.Name
	if defaultName == "" {
		base := path.Base(strings.TrimRight(src.Source, "/"))
		defaultName = strings.TrimSuffix(base, filepath.Ext(base))
	}
	skill, err := Parse(data, defaultName)
	if err != nil {
		return Skill{}, fmt.Errorf("skill %s: %w", src.Source, err)
	}
	if src.Name != "" {
		skill.Name = src.Name
	}
	skill.Path = rawURL
	return skill, nil
}

// FetchAll fetches every remote source, preserving order.
func FetchAll(ctx context.Context, srcs []RemoteSource, opts FetchOptions) ([]Skill, error) {
	out := make([]Skill, 0, len(srcs))
	for _, src := range srcs {
		s, err := FetchRemote(ctx, src, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// resolveURL turns a source (GitHub shorthand or direct URL) into a fetch URL.
func resolveURL(src RemoteSource) (string, error) {
	source := strings.TrimSpace(src.Source)
	if source == "" {
		return "", fmt.Errorf("remote skill is missing a source")
	}

	if strings.HasPrefix(source, "github.com/") {
		if strings.TrimSpace(src.Ref) == "" {
			return "", fmt.Errorf("github skill %q requires a 'ref' (commit SHA)", source)
		}
		rest := strings.TrimPrefix(source, "github.com/")
		parts := strings.SplitN(rest, "/", 3)
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return "", fmt.Errorf("github skill %q must be github.com/<org>/<repo>/<path>", source)
		}
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
			parts[0], parts[1], src.Ref, parts[2]), nil
	}

	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		return source, nil
	}

	return "", fmt.Errorf("unsupported skill source %q (use github.com/... or an https:// URL)", source)
}

func allowedHosts(opts FetchOptions) []string {
	if len(opts.AllowedHosts) > 0 {
		return opts.AllowedHosts
	}
	return DefaultAllowedHosts
}

func checkHost(rawURL string, allowed []string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid skill URL %q: %w", rawURL, err)
	}
	for _, h := range allowed {
		if strings.EqualFold(u.Hostname(), h) {
			return nil
		}
	}
	return fmt.Errorf("skill host %q is not in the allowlist %v", u.Hostname(), allowed)
}

func resolveCacheDir(dir string) (string, error) {
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			base = os.TempDir()
		}
		dir = filepath.Join(base, "agents-sdk", "skills")
	}
	// Integrity-sensitive: keep the cache private to the user.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func cacheKey(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:])
}

// readCache returns cached bytes if present and (when wantSHA is set) matching;
// it returns (nil, nil) on a miss so the caller fetches fresh.
func readCache(cachePath, wantSHA string) ([]byte, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if wantSHA != "" {
		if verifyChecksum(data, wantSHA) != nil {
			return nil, nil // stale cache; refetch
		}
	}
	return data, nil
}

func writeCache(cachePath string, data []byte) error {
	return os.WriteFile(cachePath, data, 0o600)
}

func download(ctx context.Context, rawURL string, opts FetchOptions, allowed []string) ([]byte, error) {
	client := opts.Client
	if client == nil {
		// Re-check redirect targets against the allowlist so it stays a real
		// boundary even when a server redirects cross-host.
		client = &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return checkHost(req.URL.String(), allowed)
			},
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching skill %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching skill %s: unexpected status %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
}

func verifyChecksum(data []byte, wantSHA string) error {
	if wantSHA == "" {
		return nil
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, strings.TrimSpace(wantSHA)) {
		return fmt.Errorf("checksum mismatch: got %s, want %s", got, wantSHA)
	}
	return nil
}
