package channels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// HTTPChannel exposes an agent over HTTP. POST /chat with a JSON body
// {"input": "..."} returns {"output": "..."}. GET /health returns 200.
type HTTPChannel struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
}

// NewHTTPChannel creates an HTTP channel bound to addr (defaults to ":8080").
func NewHTTPChannel(addr string) *HTTPChannel {
	if addr == "" {
		addr = ":8080"
	}
	return &HTTPChannel{Addr: addr}
}

// Name identifies the channel.
func (c *HTTPChannel) Name() string { return "http" }

type chatRequest struct {
	Input string `json:"input"`
}

type chatResponse struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Start runs the HTTP server until ctx is cancelled.
func (c *HTTPChannel) Start(ctx context.Context, handler Handler) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, chatResponse{Error: "use POST"})
			return
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, chatResponse{Error: "invalid JSON body"})
			return
		}
		if req.Input == "" {
			writeJSON(w, http.StatusBadRequest, chatResponse{Error: "'input' is required"})
			return
		}

		output, err := handler(r.Context(), req.Input)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, chatResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, chatResponse{Output: output})
	})

	server := &http.Server{
		Addr:              c.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
