package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/control"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func runFleetCLI(args []string, role string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] == "upgrade-worker" {
		fs := flag.NewFlagSet("upgrade-worker", flag.ContinueOnError)
		dir := fs.String("data-dir", "data", "data directory")
		if err := fs.Parse(args[1:]); err != nil {
			return true, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		return true, runFleetUpgradeWorker(ctx, *dir)
	}
	if args[0] == "upgrade-recover" {
		fs := flag.NewFlagSet("upgrade-recover", flag.ContinueOnError)
		dir := fs.String("data-dir", "data", "data directory")
		if err := fs.Parse(args[1:]); err != nil {
			return true, err
		}
		return true, recoverFleetUpgrade(*dir)
	}
	if (args[0] == "fleet" || args[0] == "ssl") && role == "control" {
		return true, runFleetAdminCLI(args)
	}
	if args[0] == "serve" {
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		dir := fs.String("data-dir", "data", "data directory")
		if err := fs.Parse(args[1:]); err != nil {
			return true, err
		}
		if fs.NArg() != 0 {
			return true, errors.New("unexpected serve arguments")
		}
		if err := os.Setenv("UNIFIEDPROXY_DATA_DIR", *dir); err != nil {
			return true, err
		}
		os.Args = os.Args[:1]
		return false, nil
	}
	if args[0] == "fleet-key" && role == "control" {
		fs := flag.NewFlagSet("fleet-key", flag.ContinueOnError)
		dir := fs.String("data-dir", "data", "data directory")
		if err := fs.Parse(args[1:]); err != nil {
			return true, err
		}
		var id fleet.Identity
		if err := fleet.ReadJSON(filepath.Join(*dir, "control-identity.json"), &id); err != nil {
			return true, err
		}
		fmt.Println(id.Public())
		return true, nil
	}
	if args[0] != "agent" {
		return false, nil
	}
	if role == "control" {
		return true, errors.New("agent commands are only available in node")
	}
	if len(args) < 2 {
		return true, errors.New("usage: node agent join|leave|status|relocate|break-glass|resume")
	}
	fs := flag.NewFlagSet("agent "+args[1], flag.ContinueOnError)
	dir := fs.String("data-dir", "data", "data directory")
	url := fs.String("control", "", "HTTPS control URL")
	key := fs.String("control-key", "", "trusted Ed25519 public key")
	token := fs.String("token", "", "single-use enrollment token")
	stdin := fs.Bool("token-stdin", false, "read enrollment token from stdin")
	name := fs.String("name", "", "node display name")
	if err := fs.Parse(args[2:]); err != nil {
		return true, err
	}
	if fs.NArg() != 0 {
		return true, errors.New("unexpected agent arguments")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	switch args[1] {
	case "join":
		if _, err := newPeerNetwork(*dir); err != nil {
			return true, err
		}
		if *stdin {
			b, err := io.ReadAll(io.LimitReader(os.Stdin, 4097))
			if err != nil {
				return true, err
			}
			if len(b) > 4096 {
				return true, errors.New("token too long")
			}
			*token = strings.TrimSpace(string(b))
		}
		cfg, err := agent.Enroll(ctx, *dir, *url, *key, *token, *name)
		if err == nil {
			fmt.Printf("Enrolled node %s. Restart node to activate fleet management.\n", cfg.NodeID)
		}
		return true, err
	case "status":
		cfg, err := agent.LoadConfig(*dir)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("standalone")
			return true, nil
		}
		if err != nil {
			return true, err
		}
		var applied fleet.DesiredState
		_ = fleet.ReadJSON(filepath.Join(*dir, "applied-state.json"), &applied)
		_, glass := os.Stat(filepath.Join(*dir, "break-glass"))
		return true, json.NewEncoder(os.Stdout).Encode(map[string]any{"nodeId": cfg.NodeID, "controlURL": cfg.ControlURL, "controlKey": cfg.ControlKey, "revision": applied.Revision, "digest": applied.Digest, "breakGlass": glass == nil})
	case "leave":
		if err := os.Remove(filepath.Join(*dir, agent.ConfigFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return true, err
		}
		fmt.Println("Detached locally. Revoke the identity in control; restart node before using standalone P2P features.")
		return true, nil
	case "relocate":
		return true, agent.Relocate(ctx, *dir, *url)
	case "break-glass":
		return true, fleet.WriteFile(filepath.Join(*dir, "break-glass"), []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
	case "resume":
		if _, err := agent.LoadConfig(*dir); err != nil {
			return true, err
		}
		if err := os.Remove(filepath.Join(*dir, "break-glass")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return true, err
		}
		fmt.Println("Restart node to resume fleet management.")
		return true, nil
	default:
		return true, errors.New("unknown agent command")
	}
}

func (a *App) initializeFleet(ctx context.Context, dir string) error {
	if a.role == "control" {
		store, err := control.Open(dir)
		if err != nil {
			return err
		}
		a.fleet = &fleetConsole{store: store, server: &control.Server{Store: store}, dataDir: dir, editors: map[string]*App{}}
		journalPath := filepath.Join(dir, "fleet-restore-transaction.json")
		if data, err := os.ReadFile(journalPath); err == nil {
			if err = a.restoreFleetBackupTransaction(data, false); err != nil {
				return fmt.Errorf("recover interrupted restore: %w", err)
			}
			if err = os.Remove(journalPath); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		go a.renewFleetCertificates(ctx)
		return nil
	}
	if _, err := agent.LoadConfig(dir); errors.Is(err, os.ErrNotExist) {
		return a.initializeStandalone(ctx, dir)
	} else if err != nil {
		return err
	}
	n := &nodeRuntime{manager: a.manager, sidecars: &agent.Sidecars{Dir: dir}}
	if err := n.recover(); err != nil {
		return err
	}
	a.nodeRuntime = n
	go n.supervise(ctx)
	worker, err := agent.New(dir, n)
	if err != nil {
		return err
	}
	a.fleetStart = func() { go worker.Run(ctx) }
	go func() { <-ctx.Done(); n.sidecars.Stop() }()
	return nil
}
