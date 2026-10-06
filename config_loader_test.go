// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSystemListSeparatesConfigAndSystemlist(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	systemListPath := filepath.Join(dir, "systemlist.yml")

	configContent := `
# config.yml without systemlist
venderlist:
  TestVendor:
    evapi: TEST_API_KEY
    apikey: secret
    url: https://example.com
modellist:
  demo-model:
    comment: "1"
    model: openai
    vendor: TestVendor
enckey: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	systemListContent := `
systemlist:
  NixOS: "Linux system"
  Go: "Go language"
`
	if err := os.WriteFile(systemListPath, []byte(systemListContent), 0o600); err != nil {
		t.Fatalf("WriteFile(systemlist): %v", err)
	}

	oldConfig := config
	oldSystemlist := systemlist
	oldSystemkeys := systemkeys
	defer func() {
		config = oldConfig
		systemlist = oldSystemlist
		systemkeys = oldSystemkeys
	}()

	if err := LoadConfig(configPath); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(config.Systemlist) != 0 {
		t.Fatalf("config.Systemlist should be empty before loading systemlist, got %d entries", len(config.Systemlist))
	}

	if err := LoadSystemList(systemListPath); err != nil {
		t.Fatalf("LoadSystemList() error = %v", err)
	}
	if got := systemlist["NixOS"]; got != "Linux system" {
		t.Fatalf("systemlist[NixOS] = %q, want %q", got, "Linux system")
	}
	if got := systemlist["Go"]; got != "Go language" {
		t.Fatalf("systemlist[Go] = %q, want %q", got, "Go language")
	}
	if len(config.Systemlist) != 2 {
		t.Fatalf("config.Systemlist length = %d, want 2", len(config.Systemlist))
	}
}
