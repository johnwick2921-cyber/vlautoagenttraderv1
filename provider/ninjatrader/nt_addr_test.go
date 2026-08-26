package ninjatrader

import "testing"

// NT_TCP_LISTEN_ADDR override: unset → the live default (process-singleton
// behavior byte-identical); set → a second instance can bind elsewhere.
func TestListenAddrOverride(t *testing.T) {
	if got := ListenAddr(); got != TCPListenAddr {
		t.Fatalf("unset env must yield the default %q, got %q", TCPListenAddr, got)
	}
	t.Setenv("NT_TCP_LISTEN_ADDR", "127.0.0.1:36984")
	if got := ListenAddr(); got != "127.0.0.1:36984" {
		t.Fatalf("override ignored, got %q", got)
	}
	t.Setenv("NT_TCP_LISTEN_ADDR", "   ")
	if got := ListenAddr(); got != TCPListenAddr {
		t.Fatalf("blank override must fall back to the default, got %q", got)
	}
}
