module github.com/hlonglin/unifiedproxy

go 1.24

// Release builds need the CVE-2026-111943 fix in crypto/x509 (Go 1.26.10 / 1.27.3).
toolchain go1.26.10

require gopkg.in/yaml.v3 v3.0.1
