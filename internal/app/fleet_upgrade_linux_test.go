package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestFleetUpgradeRecoveryAfterWorkerInterrupted(t *testing.T) {
	dir := t.TempDir()
	commands := filepath.Join(dir, "commands")
	shim := []byte("#!/bin/sh\nif [ \"$1\" = is-active ]; then exit 3; fi\nexit 0\n")
	if err := os.Mkdir(commands, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(commands, "systemctl"), shim, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", commands+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, phase := range []string{"scheduled", "backed-up", "verifying"} {
		t.Run(phase, func(t *testing.T) {
			exe := filepath.Join(dir, "node")
			if err := os.WriteFile(exe, []byte("candidate"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(exe+".previous", []byte("previous"), 0755); err != nil {
				t.Fatal(err)
			}
			p := fleetUpgrade{Executable: exe, Service: "unifiedproxy-node.service", Status: "pending-restart", Phase: phase, StartedAt: time.Now().Add(-11 * time.Minute), WorkerUnit: "unifiedproxy-upgrade-0123456789abcdef.service"}
			path := filepath.Join(dir, "pending-upgrade.json")
			if err := fleet.WriteJSON(path, p); err != nil {
				t.Fatal(err)
			}
			if err := recoverFleetUpgrade(dir); err != nil {
				t.Fatal(err)
			}
			if err := fleet.ReadJSON(path, &p); err != nil {
				t.Fatal(err)
			}
			expected := "rolled-back"
			if phase == "scheduled" {
				expected = "failed"
			}
			if p.Status != expected {
				t.Fatal("unsettled upgrade", p)
			}
			binary, _ := os.ReadFile(exe)
			if phase != "scheduled" && string(binary) != "previous" {
				t.Fatal("old binary not restored")
			}
		})
	}
}
