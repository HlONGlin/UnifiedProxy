// A process-level fixture for fleet apply/reload/rollback integration tests.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var failStartup string

func main() {
	dir := flag.String("d", ".", "")
	path := flag.String("f", "", "")
	test := flag.Bool("t", false, "")
	version := flag.Bool("v", false, "")
	flag.Parse()
	if *version {
		fmt.Println("Mihomo v0.0.1-test")
		return
	}
	b, err := os.ReadFile(*path)
	if err != nil {
		os.Exit(2)
	}
	var config map[string]any
	if yaml.Unmarshal(b, &config) != nil || strings.Contains(string(b), "reject-me") {
		os.Exit(3)
	}
	if *test {
		return
	}
	if failStartup == "1" {
		os.Exit(5)
	}
	address, _ := config["external-controller"].(string)
	secret, _ := config["secret"].(string)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			w.WriteHeader(401)
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/configs" {
			if _, err := os.Stat(filepath.Join(*dir, "fail-reload")); err == nil {
				_ = os.Remove(filepath.Join(*dir, "fail-reload"))
				w.WriteHeader(500)
				return
			}
			var p struct {
				Path string `json:"path"`
			}
			if json.NewDecoder(r.Body).Decode(&p) != nil {
				w.WriteHeader(400)
				return
			}
			if _, err := os.Stat(p.Path); err != nil {
				w.WriteHeader(400)
				return
			}
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"test","connections":[],"uploadTotal":0,"downloadTotal":0}`))
	})
	if http.ListenAndServe(address, handler) != nil {
		os.Exit(4)
	}
}
