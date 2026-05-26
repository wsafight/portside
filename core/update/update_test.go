package update

import (
	"path/filepath"
	"testing"

	"portside/core/porthome"
)

func TestCheckUsesDefaultConfigWhenHomeMissing(t *testing.T) {
	t.Setenv("PORTSIDE_HOME", filepath.Join(t.TempDir(), "home"))

	result := Check("1.2.3")
	if result.CurrentVersion != "1.2.3" {
		t.Fatalf("version = %s", result.CurrentVersion)
	}
	if result.Channel != "stable" {
		t.Fatalf("channel = %s", result.Channel)
	}
	if result.Status != "not_checked" {
		t.Fatalf("status = %s", result.Status)
	}
	if len(result.Endpoints) == 0 {
		t.Fatal("expected default endpoints")
	}
}

func TestCheckUsesSavedUpdateConfig(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	if _, err := porthome.Init(); err != nil {
		t.Fatal(err)
	}

	config, err := porthome.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	config.Update.Channel = "nightly"
	config.Update.Source = "manual"
	config.Update.Endpoints = []porthome.UpdateEndpoint{{Name: "test", Manifest: "https://example.invalid/manifest.json"}}
	if err := porthome.SaveConfig(config); err != nil {
		t.Fatal(err)
	}

	result := Check("dev")
	if result.Channel != "nightly" {
		t.Fatalf("channel = %s", result.Channel)
	}
	if result.Source != "manual" {
		t.Fatalf("source = %s", result.Source)
	}
	if len(result.Endpoints) != 1 || result.Endpoints[0].Name != "test" {
		t.Fatalf("endpoints = %#v", result.Endpoints)
	}
}
