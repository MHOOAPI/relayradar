package favicon

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveFavicon is opt-in: LIVE_FAVICON=1 go test ./internal/favicon -run TestLiveFavicon -v
func TestLiveFavicon(t *testing.T) {
	if os.Getenv("LIVE_FAVICON") == "" {
		t.Skip("set LIVE_FAVICON=1 to hit real hosts")
	}
	f := New(true)
	for _, d := range []string{"aihubmix.com", "302.ai", "cun.ai", "cubence.com", "wenming7.cn"} {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		data, ct, ok := f.Get(ctx, d)
		cancel()
		t.Logf("%-14s ok=%v len=%d ctype=%s", d, ok, len(data), ct)
	}
}
