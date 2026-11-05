package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

// These release pins are preserved from UnifiedProxy's sidecar installers.
func sidecarAsset(kind, arch string) (source, binary string, archive bool, err error) {
	switch kind {
	case "naive":
		if arch != "amd64" {
			return "", "", false, errors.New("NaiveProxy requires a locally provisioned caddy-naive binary on this architecture")
		}
		return "https://github.com/klzgrad/forwardproxy/releases/download/v2.11.7-naive2/caddy-forwardproxy-naive.tar.xz", "caddy", true, nil
	case "shadowtls":
		target := map[string]string{"amd64": "x86_64-unknown-linux-musl", "arm64": "aarch64-unknown-linux-musl", "arm": "armv7-unknown-linux-musleabihf"}[arch]
		if target == "" {
			return "", "", false, errors.New("unsupported ShadowTLS architecture")
		}
		return "https://github.com/ihciah/shadow-tls/releases/download/v0.2.25/shadow-tls-" + target, "shadow-tls", false, nil
	case "shadowsocks":
		target := map[string]string{"amd64": "x86_64-unknown-linux-musl", "arm64": "aarch64-unknown-linux-musl", "arm": "armv7-unknown-linux-musleabihf"}[arch]
		if target == "" {
			return "", "", false, errors.New("unsupported ssserver architecture")
		}
		return "https://github.com/shadowsocks/shadowsocks-rust/releases/download/v1.18.2/shadowsocks-v1.18.2." + target + ".tar.xz", "ssserver", true, nil
	}
	return "", "", false, errors.New("unknown sidecar kind")
}

type limitedOutput struct {
	writer    io.Writer
	remaining int64
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errors.New("sidecar binary exceeds size limit")
	}
	count, err := w.writer.Write(p)
	w.remaining -= int64(count)
	return count, err
}

func (s *Sidecars) ensure(ctx context.Context, kind string) error {
	name := map[string]string{"naive": "caddy-naive", "shadowtls": "shadow-tls", "shadowsocks": "ssserver"}[kind]
	if name == "" {
		return errors.New("unknown sidecar")
	}
	destination := s.binary(name)
	if info, err := os.Stat(destination); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		return nil
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("provision %s locally at %s", name, destination)
	}
	source, member, archive, err := sidecarAsset(kind, runtime.GOARCH)
	if err != nil {
		return err
	}
	job, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(job, "GET", source, nil)
	if err != nil {
		return err
	}
	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 10 {
			return errors.New("invalid sidecar download redirect")
		}
		return nil
	}}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("sidecar download HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (96<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 96<<20 {
		return errors.New("sidecar archive too large")
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	if archive {
		file, err := os.CreateTemp(s.Dir, ".sidecar-*.tar.xz")
		if err != nil {
			return err
		}
		archivePath := file.Name()
		defer os.Remove(archivePath)
		_, err = file.Write(data)
		_ = file.Close()
		if err != nil {
			return err
		}
		var listingBuffer bytes.Buffer
		var listingErrors bytes.Buffer
		listCommand := exec.CommandContext(job, "tar", "-tJf", archivePath)
		listCommand.Stdout = &limitedOutput{writer: &listingBuffer, remaining: 1 << 20}
		listCommand.Stderr = &limitedOutput{writer: &listingErrors, remaining: 4096}
		err = listCommand.Run()
		listing := listingBuffer.Bytes()
		if err != nil {
			return fmt.Errorf("inspect sidecar archive (tar and xz required): %w: %s", err, strings.TrimSpace(listingErrors.String()))
		}
		entry := ""
		for _, line := range strings.Split(string(listing), "\n") {
			candidate := path.Clean(strings.TrimSpace(line))
			if path.Base(candidate) == member && !path.IsAbs(candidate) && !strings.HasPrefix(candidate, "../") {
				entry = strings.TrimSpace(line)
				break
			}
		}
		if entry == "" {
			return errors.New("binary missing from sidecar archive")
		}
		output, err := os.CreateTemp(filepath.Dir(destination), ".sidecar-bin-*")
		if err != nil {
			return err
		}
		tmp := output.Name()
		defer os.Remove(tmp)
		cmd := exec.CommandContext(job, "tar", "-xOJf", archivePath, "--", entry)
		cmd.Stdout = &limitedOutput{writer: output, remaining: 128 << 20}
		cmd.Stderr = io.Discard
		err = cmd.Run()
		closeErr := output.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		data, err = os.ReadFile(tmp)
		if err != nil {
			return err
		}
	}
	if err = fleet.WriteFile(destination, data, 0755); err != nil {
		return err
	}
	if kind == "naive" {
		modules, err := exec.CommandContext(job, destination, "list-modules").Output()
		if err != nil || !strings.Contains(string(modules), "http.handlers.forward_proxy") {
			_ = os.Remove(destination)
			return errors.New("Caddy is missing its NaiveProxy module")
		}
	}
	sum := sha256.Sum256(data)
	return fleet.WriteJSON(destination+".source.json", map[string]string{"source": source, "sha256": hex.EncodeToString(sum[:])})
}
