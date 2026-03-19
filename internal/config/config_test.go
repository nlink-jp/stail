package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/magifd2/stail/internal/config"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &config.Config{
		CurrentProfile: "work",
		Profiles: map[string]config.Profile{
			"work": {
				Provider: config.ProviderSlack,
				Token:    "xoxb-test",
				AppToken: "xapp-test",
				Channel:  "#general",
			},
		},
	}

	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file perm = %o, want 0600", got)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.CurrentProfile != cfg.CurrentProfile {
		t.Errorf("CurrentProfile = %q, want %q", loaded.CurrentProfile, cfg.CurrentProfile)
	}
	p, err := loaded.GetProfile("")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if p.Token != "xoxb-test" {
		t.Errorf("Token = %q, want %q", p.Token, "xoxb-test")
	}
	if p.AppToken != "xapp-test" {
		t.Errorf("AppToken = %q, want %q", p.AppToken, "xapp-test")
	}
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "config.json")

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save in nested dir: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestGetProfile(t *testing.T) {
	cfg := &config.Config{
		CurrentProfile: "a",
		Profiles: map[string]config.Profile{
			"a": {Token: "tok-a"},
			"b": {Token: "tok-b"},
		},
	}

	t.Run("empty name uses current", func(t *testing.T) {
		p, err := cfg.GetProfile("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Token != "tok-a" {
			t.Errorf("Token = %q, want tok-a", p.Token)
		}
	})

	t.Run("named profile", func(t *testing.T) {
		p, err := cfg.GetProfile("b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Token != "tok-b" {
			t.Errorf("Token = %q, want tok-b", p.Token)
		}
	})

	t.Run("missing profile returns error", func(t *testing.T) {
		_, err := cfg.GetProfile("missing")
		if err == nil {
			t.Error("expected error for missing profile, got nil")
		}
	})
}

func TestDetectServerMode(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		t.Setenv("STAIL_MODE", "")
		ok, err := config.DetectServerMode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false when STAIL_MODE is empty")
		}
	})

	t.Run("server mode", func(t *testing.T) {
		t.Setenv("STAIL_MODE", "server")
		ok, err := config.DetectServerMode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true when STAIL_MODE=server")
		}
	})

	t.Run("invalid value returns error", func(t *testing.T) {
		t.Setenv("STAIL_MODE", "invalid")
		_, err := config.DetectServerMode()
		if err == nil {
			t.Error("expected error for invalid STAIL_MODE value")
		}
	})
}

func TestBuildConfigFromEnv(t *testing.T) {
	t.Run("missing STAIL_PROVIDER", func(t *testing.T) {
		t.Setenv("STAIL_PROVIDER", "")
		t.Setenv("STAIL_TOKEN", "xoxb-test")
		_, err := config.BuildConfigFromEnv()
		if err == nil {
			t.Error("expected error when STAIL_PROVIDER is missing")
		}
	})

	t.Run("missing STAIL_TOKEN", func(t *testing.T) {
		t.Setenv("STAIL_PROVIDER", "slack")
		t.Setenv("STAIL_TOKEN", "")
		_, err := config.BuildConfigFromEnv()
		if err == nil {
			t.Error("expected error when STAIL_TOKEN is missing")
		}
	})

	t.Run("valid env vars", func(t *testing.T) {
		t.Setenv("STAIL_PROVIDER", "slack")
		t.Setenv("STAIL_TOKEN", "xoxb-abc")
		t.Setenv("STAIL_APP_TOKEN", "xapp-xyz")
		t.Setenv("STAIL_CHANNEL", "#test")

		cfg, err := config.BuildConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p, err := cfg.GetProfile("")
		if err != nil {
			t.Fatalf("GetProfile: %v", err)
		}
		if p.Provider != "slack" {
			t.Errorf("Provider = %q, want slack", p.Provider)
		}
		if p.Token != "xoxb-abc" {
			t.Errorf("Token = %q, want xoxb-abc", p.Token)
		}
		if p.AppToken != "xapp-xyz" {
			t.Errorf("AppToken = %q, want xapp-xyz", p.AppToken)
		}
		if p.Channel != "#test" {
			t.Errorf("Channel = %q, want #test", p.Channel)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.CurrentProfile == "" {
		t.Error("CurrentProfile should not be empty")
	}
	if len(cfg.Profiles) == 0 {
		t.Error("Profiles should not be empty")
	}
}
