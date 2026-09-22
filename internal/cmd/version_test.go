package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The Homebrew formula tests `stail --version`, and `make verify-release`
// requires the packaged binary's --version to contain the tag. stail answered
// neither form until v0.6.0: main.version was injected and never used.
func TestVersionFlagAndSubcommandPrintTheSameLine(t *testing.T) {
	// A config file that does not parse: loading it fails the root's pre-run
	// hook, so both forms passing proves neither reads the config.
	bad := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STAIL_MODE", "")

	oldVersion := rootCmd.Version
	rootCmd.Version = "v9.9.9"
	t.Cleanup(func() {
		rootCmd.Version = oldVersion
		flagConfig = ""
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	var outputs []string
	for _, args := range [][]string{{"--version"}, {"version"}, {"--config", bad, "version"}} {
		var out bytes.Buffer
		rootCmd.SetOut(&out)
		rootCmd.SetErr(&out)
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%v: %v (output %q)", args, err, out.String())
		}
		outputs = append(outputs, out.String())
	}
	want := "stail version v9.9.9\n"
	for i, got := range outputs {
		if got != want {
			t.Errorf("form %d printed %q, want %q", i, got, want)
		}
	}
}
