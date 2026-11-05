package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

// Administration always goes through the live panel. Opening the durable store
// in a second process would race the server's in-memory state.
func runFleetAdminCLI(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: control fleet nodes|token|revoke|publish|tasks|task|batch|import|backup|restore|subscriptions|audit|update; control ssl list|issue|renew|import|bind [--url PANEL_URL --username USER --password-stdin --json FILE]")
	}
	group, action := args[0], args[1]
	fs := flag.NewFlagSet(group+" "+action, flag.ContinueOnError)
	panel := fs.String("url", "", "panel HTTPS URL including panel path")
	user := fs.String("username", "", "panel username")
	passwordStdin := fs.Bool("password-stdin", false, "read panel password from stdin")
	file := fs.String("json", "", "JSON input file")
	nodeID := fs.String("node", "", "node ID")
	name := fs.String("name", "", "node or certificate name")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected CLI arguments")
	}
	if err := fleet.ValidateControlURL(*panel); err != nil {
		return err
	}
	if *user == "" || !*passwordStdin {
		return errors.New("--username and --password-stdin are required")
	}
	secret, err := io.ReadAll(io.LimitReader(os.Stdin, 4097))
	if err != nil {
		return err
	}
	if len(secret) > 4096 {
		return errors.New("password too long")
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 12 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	base := strings.TrimRight(*panel, "/")
	call := func(method, path string, body []byte) ([]byte, error) {
		req, err := http.NewRequest(method, base+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		data, err := io.ReadAll(io.LimitReader(res.Body, (64<<20)+1))
		if err != nil {
			return nil, err
		}
		if len(data) > 64<<20 {
			return nil, errors.New("response too large")
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			var failure struct{ Message string }
			_ = json.Unmarshal(data, &failure)
			return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, failure.Message)
		}
		return data, nil
	}
	login, _ := json.Marshal(map[string]string{"username": *user, "password": strings.TrimRight(string(secret), string([]byte{13, 10}))})
	if _, err = call("POST", "/api/auth/login", login); err != nil {
		return err
	}
	method, path := "GET", "/api/fleet"
	var body []byte
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			return err
		}
		body, err = io.ReadAll(io.LimitReader(f, (64<<20)+1))
		_ = f.Close()
		if err != nil {
			return err
		}
		if len(body) > 64<<20 || !json.Valid(body) {
			return errors.New("invalid or oversized JSON input")
		}
	}
	requireNode := func() error {
		if !fleet.SafeName(*nodeID) {
			return errors.New("--node is required")
		}
		return nil
	}
	if group == "ssl" {
		path += "/certificates"
		if action != "list" {
			method = "POST"
			path += "/" + action
			if body == nil {
				body, _ = json.Marshal(map[string]string{"name": *name})
			}
		}
	} else {
		switch action {
		case "nodes":
			if *nodeID != "" {
				if err = requireNode(); err != nil {
					return err
				}
				path += "/nodes/" + *nodeID
			}
		case "token":
			method = "POST"
			path += "/tokens"
			body, _ = json.Marshal(map[string]string{"name": *name, "nodeId": *nodeID})
		case "revoke", "publish":
			if err = requireNode(); err != nil {
				return err
			}
			method = "POST"
			path += "/nodes/" + *nodeID + "/" + action
			body = []byte("{}")
		case "tasks", "subscriptions", "audit":
			path += "/" + action
		case "task":
			method = "POST"
			path += "/tasks"
		case "batch", "update":
			method = "POST"
			path += "/" + action
		case "import":
			method = "POST"
			path += "/import/merit"
		case "backup":
			path += "/backup"
		case "restore":
			method = "POST"
			path += "/backup"
			if body != nil {
				body, err = json.Marshal(map[string]any{"action": "restore", "backup": json.RawMessage(body)})
				if err != nil {
					return err
				}
			}
		default:
			return errors.New("unknown fleet command")
		}
	}
	if method == "POST" && body == nil {
		return errors.New("--json FILE is required")
	}
	data, err := call(method, path, body)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
