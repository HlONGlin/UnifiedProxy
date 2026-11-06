package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

// Manual stop is local runtime intent, separate from the controller's policy.
// Internal shutdowns and upgrades must not change it.
func (m *CoreManager) coreStopped() (bool, error) {
	_, err := os.Stat(filepath.Join(m.dataDir, "core-stopped"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (m *CoreManager) setCoreStopped(stopped bool) error {
	path := filepath.Join(m.dataDir, "core-stopped")
	if stopped {
		return fleet.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *CoreManager) controlCore(action string) error {
	m.coreMu.Lock()
	defer m.coreMu.Unlock()
	if action != "start" && action != "stop" && action != "restart" {
		return errors.New("unknown core action")
	}
	previous, err := m.coreStopped()
	if err != nil {
		return err
	}
	if err = m.setCoreStopped(action == "stop"); err != nil {
		return fmt.Errorf("save core run state: %w", err)
	}
	switch action {
	case "start":
		err = m.startCoreLocked()
	case "stop":
		err = m.stopCoreLocked()
	case "restart":
		err = m.stopCoreLocked()
		if err == nil {
			err = m.startCoreLocked()
		}
	}
	if err != nil {
		if restoreErr := m.setCoreStopped(previous); restoreErr != nil {
			return fmt.Errorf("%v; restore core run state: %w", err, restoreErr)
		}
	}
	return err
}
