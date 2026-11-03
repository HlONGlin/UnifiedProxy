package fleet

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFleetReplayPersistenceAndCorruptionFailClosed(t *testing.T) {
	identity, _ := NewIdentity()
	message, _ := Sign(identity, "node", "POST /test", map[string]bool{"ok": true})
	path := filepath.Join(t.TempDir(), "replays.json")
	guard := ReplayGuard{File: path}
	if err := guard.Verify(message, identity.Public(), "node", "POST /test", time.Now()); err != nil {
		t.Fatal(err)
	}
	restarted := ReplayGuard{File: path}
	if err := restarted.Verify(message, identity.Public(), "node", "POST /test", time.Now()); err == nil {
		t.Fatal("replay accepted after restart")
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	broken := ReplayGuard{File: path}
	for i := 0; i < 2; i++ {
		if err := broken.Verify(message, identity.Public(), "node", "POST /test", time.Now()); err == nil {
			t.Fatal("corrupt replay protection failed open")
		}
	}
}
