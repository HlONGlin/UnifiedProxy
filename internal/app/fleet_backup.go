package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type fleetBackup struct {
	Format    int                        `json:"format"`
	CreatedAt time.Time                  `json:"createdAt"`
	Fleet     json.RawMessage            `json:"fleet"`
	Panel     State                      `json:"panel"`
	Files     map[string]json.RawMessage `json:"files"`
	Drafts    map[string]State           `json:"drafts,omitempty"`
}

var fleetBackupFiles = []string{"sub-templates.json", "multi-control.json", "client-devices.json", "cross-panel-subscriptions.json", fleetCertificateFile}

func (a *App) exportFleetBackup() ([]byte, error) {
	f := a.fleet
	data, err := f.store.Backup()
	if err != nil {
		return nil, err
	}
	a.manager.mu.Lock()
	state := copyState(a.manager.state)
	a.manager.mu.Unlock()
	b := fleetBackup{Format: 1, CreatedAt: time.Now().UTC(), Fleet: data, Panel: state, Files: map[string]json.RawMessage{}, Drafts: map[string]State{}}
	for _, node := range f.store.Nodes() {
		if editor := f.editors[node.ID]; editor != nil {
			editor.manager.mu.Lock()
			b.Drafts[node.ID] = copyState(editor.manager.state)
			editor.manager.mu.Unlock()
			continue
		}
		var draft State
		if err := fleet.ReadJSON(filepath.Join(f.dataDir, "fleet-nodes", node.ID, "state.json"), &draft); err == nil {
			b.Drafts[node.ID] = draft
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read node draft %s: %w", node.ID, err)
		}
	}
	for _, name := range fleetBackupFiles {
		data, err := os.ReadFile(filepath.Join(f.dataDir, name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !json.Valid(data) {
			return nil, fmt.Errorf("invalid backup file %s", name)
		}
		b.Files[name] = data
	}
	return json.Marshal(b)
}
func (a *App) restoreFleetBackup(data []byte) error {
	return a.restoreFleetBackupTransaction(data, true)
}
func (a *App) restoreFleetBackupTransaction(data []byte, journal bool) error {
	if len(data) > 64<<20 {
		return errors.New("fleet backup too large")
	}
	var b fleetBackup
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	if b.Format != 1 || b.Panel.Settings.PanelListen == "" || b.Panel.Settings.Username == "" || b.Panel.Settings.Password == "" {
		return errors.New("invalid panel in fleet backup")
	}
	if _, err := renderStateConfig(b.Panel); err != nil {
		return err
	}
	for name, value := range b.Files {
		allowed := false
		for _, candidate := range fleetBackupFiles {
			if name == candidate {
				allowed = true
			}
		}
		if !allowed || !json.Valid(value) {
			return errors.New("unexpected file in fleet backup")
		}
	}
	var registry struct {
		State struct {
			Nodes map[string]json.RawMessage `json:"nodes"`
		} `json:"state"`
	}
	if err := json.Unmarshal(b.Fleet, &registry); err != nil {
		return err
	}
	for id, draft := range b.Drafts {
		if !fleet.SafeName(id) || registry.State.Nodes[id] == nil {
			return errors.New("unexpected node draft in backup")
		}
		if _, err := renderStateConfig(draft); err != nil {
			return fmt.Errorf("invalid node draft %s: %w", id, err)
		}
	}
	f := a.fleet
	var oldFull []byte
	var err error
	journalPath := filepath.Join(f.dataDir, "fleet-restore-transaction.json")
	if journal {
		oldFull, err = a.exportFleetBackup()
		if err != nil {
			return err
		}
		if err = fleet.WriteFile(journalPath, oldFull, 0600); err != nil {
			return err
		}
	}
	rollback := func(cause error) error {
		if journal {
			if err := a.restoreFleetBackupTransaction(oldFull, false); err != nil {
				return fmt.Errorf("restore: %v; rollback: %w", cause, err)
			}
			_ = os.Remove(journalPath)
		}
		return cause
	}
	if err = f.store.Restore(b.Fleet); err != nil {
		if journal {
			_ = os.Remove(journalPath)
		}
		return err
	}
	// Clear drafts absent from the backup too, including nodes removed by a
	// restore. Only known state files are touched; never delete node directories.
	draftIDs := map[string]bool{}
	entries, readErr := os.ReadDir(filepath.Join(f.dataDir, "fleet-nodes"))
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return rollback(readErr)
	}
	for _, entry := range entries {
		if entry.IsDir() && fleet.SafeName(entry.Name()) {
			draftIDs[entry.Name()] = true
		}
	}
	for id := range b.Drafts {
		draftIDs[id] = true
	}
	for id := range draftIDs {
		path := filepath.Join(f.dataDir, "fleet-nodes", id, "state.json")
		if draft, ok := b.Drafts[id]; ok {
			err = fleet.WriteJSON(path, draft)
		} else {
			err = os.Remove(path)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		}
		if err != nil {
			return rollback(err)
		}
	}
	for _, name := range fleetBackupFiles {
		if value, ok := b.Files[name]; ok {
			err = fleet.WriteFile(filepath.Join(f.dataDir, name), value, 0600)
		} else {
			err = os.Remove(filepath.Join(f.dataDir, name))
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		}
		if err != nil {
			return rollback(err)
		}
	}
	a.manager.mu.Lock()
	previous := a.manager.state
	a.manager.state = b.Panel
	err = a.manager.saveLocked()
	if err != nil {
		a.manager.state = previous
	}
	a.manager.mu.Unlock()
	if err != nil {
		return rollback(err)
	}
	f.editors = map[string]*App{}
	if journal {
		if err = os.Remove(journalPath); err != nil {
			return rollback(err)
		}
	}
	return nil
}

type webDAVConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func webDAVRequest(ctx context.Context, method string, cfg webDAVConfig, checksum bool, body []byte) ([]byte, error) {
	if err := fleet.ValidateControlURL(cfg.URL); err != nil {
		return nil, err
	}
	url := cfg.URL
	if checksum {
		url += ".sha256"
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)
	if method == "PUT" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("WebDAV redirects are disabled") }}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("WebDAV HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<20)+1))
	if len(data) > 64<<20 {
		return nil, errors.New("WebDAV backup too large")
	}
	return data, err
}
func (a *App) handleFleetBackup(w http.ResponseWriter, r *http.Request) {
	f := a.fleet
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Method == "GET" {
		data, err := a.exportFleetBackup()
		if err != nil {
			writeJSON(w, 500, apiResponse{Message: err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Disposition", `attachment; filename="fleet-backup.json"`)
		_, _ = w.Write(data)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var p struct {
		Action string          `json:"action"`
		Backup json.RawMessage `json:"backup"`
		WebDAV webDAVConfig    `json:"webdav"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&p); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	var err error
	switch p.Action {
	case "restore":
		err = a.restoreFleetBackup(p.Backup)
	case "upload":
		var data []byte
		data, err = a.exportFleetBackup()
		if err == nil {
			_, err = webDAVRequest(r.Context(), "PUT", p.WebDAV, false, data)
		}
		if err == nil {
			sum := sha256.Sum256(data)
			_, err = webDAVRequest(r.Context(), "PUT", p.WebDAV, true, []byte(hex.EncodeToString(sum[:])))
		}
	case "download":
		var data, checksum []byte
		data, err = webDAVRequest(r.Context(), "GET", p.WebDAV, false, nil)
		if err == nil {
			checksum, err = webDAVRequest(r.Context(), "GET", p.WebDAV, true, nil)
		}
		if err == nil {
			sum := sha256.Sum256(data)
			if !strings.EqualFold(hex.EncodeToString(sum[:]), strings.TrimSpace(string(checksum))) {
				err = errors.New("WebDAV checksum mismatch")
			}
		}
		if err == nil {
			err = a.restoreFleetBackup(data)
		}
	default:
		err = errors.New("unknown backup action")
	}
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	writeJSON(w, 200, apiResponse{OK: true, Message: "completed; restart control after restoring to reload templates and device identities"})
}
