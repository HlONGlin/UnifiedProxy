package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestWipeProcessesHaltsCoreAndRejectsLaterApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("mode: rule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := &CoreManager{dataDir: dir, connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
	n := &nodeRuntime{manager: m, sidecars: &agent.Sidecars{Dir: dir}, applied: &fleet.DesiredState{NodeID: "node", Revision: 1}}
	if err := n.WipeProcesses(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.startCore(); err != nil {
		t.Fatal(err)
	}
	if err := m.restartCore(); err != nil {
		t.Fatal(err)
	}
	if m.runningLocked() {
		t.Fatal("proxy still running after wipe")
	}
	err := n.Apply(context.Background(), fleet.DesiredState{NodeID: "node", Revision: 2, MihomoYAML: "mode: rule", Model: json.RawMessage(`{}`)})
	if err == nil {
		t.Fatal("revoked node accepted a new configuration")
	}
}
