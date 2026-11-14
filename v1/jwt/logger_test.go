package jwt

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/contrib/v1/jwt/testutil"
)

// syncBuffer is a bytes.Buffer that is safe to write from the background
// refresh goroutines while the test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newBufferLogger() (*slog.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

// TestNew_WithLogger_WiresAllComponents verifies the logger passed with
// WithLogger reaches the middleware, the validator, and the JWKS provider.
func TestNew_WithLogger_WiresAllComponents(t *testing.T) {
	logger, _ := newBufferLogger()

	mw, err := New(
		WithJWKSEndpoint("https://auth.example.com/.well-known/jwks.json"),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if mw.Logger() != logger {
		t.Fatal("middleware should use the logger passed with WithLogger")
	}
	if mw.validator.logger != logger {
		t.Fatal("validator should use the logger passed with WithLogger")
	}
	provider, ok := mw.validator.keyProvider.(*JWKSKeyProvider)
	if !ok {
		t.Fatalf("expected JWKSKeyProvider, got %T", mw.validator.keyProvider)
	}
	if provider.logger != logger {
		t.Fatal("JWKS provider should use the logger passed with WithLogger")
	}
}

// TestNew_WithLogger_DefaultAndNil verifies slog.Default() is used when no
// logger is given, and that a nil logger does not replace an earlier one.
func TestNew_WithLogger_DefaultAndNil(t *testing.T) {
	secret := WithSecret([]byte("secret"))

	mw, err := New(secret)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if mw.Logger() != slog.Default() {
		t.Fatal("expected slog.Default() when WithLogger is not used")
	}

	mw, err = New(secret, WithLogger(nil))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if mw.Logger() != slog.Default() {
		t.Fatal("expected slog.Default() when WithLogger(nil) is used")
	}

	logger, _ := newBufferLogger()
	mw, err = New(secret, WithLogger(logger), WithLogger(nil))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if mw.Logger() != logger {
		t.Fatal("WithLogger(nil) must not replace a logger set earlier")
	}
}

// TestNew_WithLogger_SurvivesLaterSetDefault covers the case where the
// application builds the middleware first and replaces the default logger
// afterwards. The startup lines and the JWKS refresh warning must go to the
// logger passed with WithLogger, and nothing may go to the default logger that
// was current when New was called.
func TestNew_WithLogger_SurvivesLaterSetDefault(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	// The default logger at the time New is called.
	earlyDefault, earlyBuf := newBufferLogger()
	slog.SetDefault(earlyDefault)

	_, pub := testutil.GenerateRSATestKeyPair()
	srv := newJWKSTestServer(&testutil.MockJWKSKey{Kid: "key1", PublicKey: pub})
	defer srv.Close()

	logger, buf := newBufferLogger()
	mw, err := New(
		WithJWKSEndpoint(srv.URL),
		WithJWKSMinRefreshInterval(-1),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// The application installs its final default logger after New.
	lateDefault, _ := newBufferLogger()
	slog.SetDefault(lateDefault)

	ctx := context.Background()
	if err := mw.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = mw.Stop(ctx) }) //nolint:errcheck // test cleanup

	waitFor(t, 2*time.Second, func() bool {
		return strings.Contains(buf.String(), "background JWKS refresh started")
	}, "background refresh start line not logged to the WithLogger logger")

	// Endpoint fails and the cache goes stale: the provider serves the stale
	// key and reports the failed refresh through the logger.
	srv.setFail(true)
	expireCache(mw.jwksCache)

	provider, ok := mw.validator.keyProvider.(*JWKSKeyProvider)
	if !ok {
		t.Fatalf("expected JWKSKeyProvider, got %T", mw.validator.keyProvider)
	}
	if key, err := provider.GetKey(ctx, "key1"); err != nil || key == nil {
		t.Fatalf("stale key should be served, got key=%v err=%v", key, err)
	}

	waitFor(t, 2*time.Second, func() bool {
		return strings.Contains(buf.String(), "background JWKS refresh failed; serving stale keys")
	}, "stale-key warning not logged to the WithLogger logger")

	out := buf.String()
	for _, line := range []string{
		"starting JWT middleware in JWKS mode",
		"initial JWKS fetch succeeded",
		"background JWKS refresh started",
		"background JWKS refresh failed; serving stale keys",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("expected %q in the WithLogger logger output, got:\n%s", line, out)
		}
	}

	if got := earlyBuf.String(); got != "" {
		t.Errorf("nothing should be logged to the default logger captured at New, got:\n%s", got)
	}
}
