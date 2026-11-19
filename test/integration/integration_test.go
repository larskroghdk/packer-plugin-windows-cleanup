//go:build integration

// Package integration tests only explicitly authorized disposable guests.
package integration

import (
	"context"
	"encoding/json"
	"github.com/larskroghdk/packer-plugin-windows-cleanup/internal/report"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestDisposableGuest(t *testing.T) {
	host := os.Getenv("WC_TEST_HOST")
	if host == "" {
		t.Skip("set WC_TEST_HOST for a disposable VM")
	}
	apply := os.Getenv("WC_TEST_ALLOW_MUTATION") == "yes"
	dir := t.TempDir()
	binary := filepath.Join(dir, "packer-plugin-windows-cleanup")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	env := append(os.Environ(), "PACKER_PLUGIN_PATH="+filepath.Join(dir, "plugins"))
	run := func(name string, args ...string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", name, err, out)
		}
	}
	run("go", "build", "-ldflags", "-X main.versionPrerelease=", "-o", binary, "../..")
	run("packer", "plugins", "install", "--path", binary, "github.com/larskroghdk/windows-cleanup")
	username := os.Getenv("WC_TEST_USERNAME")
	if username == "" {
		username = "Administrator"
	}
	if os.Getenv("WC_TEST_PASSWORD") == "" {
		t.Fatal("WC_TEST_PASSWORD required")
	}
	rounds := []bool{true}
	if apply {
		rounds = append(rounds, false, false)
	}
	for i, dry := range rounds {
		path := filepath.Join(dir, strconv.Itoa(i)+".json")
		// Use an environment variable rather than putting credentials in process args.
		env = append(env, "PKR_VAR_host="+host, "PKR_VAR_username="+username, "PKR_VAR_password="+os.Getenv("WC_TEST_PASSWORD"), "PKR_VAR_use_ssl="+strconv.FormatBool(os.Getenv("WC_TEST_USE_SSL") != "false"))
		run("packer", "build", "-color=false", "-var", "dry_run="+strconv.FormatBool(dry), "-var", "report_path="+filepath.ToSlash(path), ".")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var r report.Report
		if err := json.Unmarshal(data, &r); err != nil {
			t.Fatal(err)
		}
		if r.Status != "success" || r.After == nil || !r.After.WinRM {
			t.Fatalf("invalid postflight: %+v", r)
		}
		if i == 2 {
			for _, op := range r.Operations {
				if len(op.Removed) != 0 {
					t.Fatal("second apply was not idempotent")
				}
			}
		}
		if apply && i == 0 {
			found := false
			for _, op := range r.Operations {
				if len(op.Candidates) > 0 {
					found = true
				}
			}
			if !found {
				t.Fatal("seed integration-owned.txt before running mutation tests")
			}
		}
	}
}
