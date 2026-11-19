package provisioner

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/larskroghdk/packer-plugin-windows-cleanup/internal/report"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSafeDefaultsAndSelection(t *testing.T) {
	var p Provisioner
	if err := p.Prepare(map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if got := p.config.operations(); !reflect.DeepEqual(got, []string{"temporary_files", "validate_system"}) {
		t.Fatal(got)
	}
	if p.config.FailOnError.False() || p.config.ResetComponentBase || p.config.Sysprep {
		t.Fatal("unsafe defaults")
	}
	var custom Provisioner
	if err := custom.Prepare(map[string]any{"profile": "custom", "component_store": true, "windows_update": true, "installer_cleanup": true, "provisioning_artifacts": true}); err != nil {
		t.Fatal(err)
	}
	if got := custom.config.operations(); !reflect.DeepEqual(got, []string{"installer_cleanup", "provisioning_artifacts", "component_store", "validate_system"}) {
		t.Fatal(got)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, raw := range []map[string]any{
		{"operation_timeout": "-1s"}, {"minimum_age_hours": -1}, {"unsupported_policy": "ignore"}, {"pending_reboot_policy": "ignore"},
		{"reset_component_base": true}, {"remove_user_profiles": true}, {"clear_event_logs": true}, {"profile": "fast"},
		{"sysprep": true}, {"sysprep": true, "packer_builder_type": "qemu", "validate_system": false},
		{"clear_event_logs": true, "event_logs": []string{"Security"}, "log_export_path": `C:\Exports`},
		{"artifact_paths": []string{`C:\Windows\Temp\..\System32`}}, {"build_profile_sids": []string{"S-1-5-18"}},
	} {
		var p Provisioner
		if err := p.Prepare(raw); err == nil {
			t.Errorf("accepted %#v", raw)
		}
	}
}

func TestWindowsPaths(t *testing.T) {
	for _, path := range []string{`C:\Windows\Temp\build`, `D:\Temp\one's file.log`} {
		if err := validatePath(path); err != nil {
			t.Error(err)
		}
	}
	for _, path := range []string{`C:\`, `C:\a\..\b`, `C:\a\b.`, `C:\a\ b `, `\\server\share\x`, `\\?\C:\Temp`, `C:\a:file`, `C:\a\NUL.txt`, `C:\a\\b`, `C:\a\*`, `relative`} {
		if validatePath(path) == nil {
			t.Errorf("accepted %s", path)
		}
	}
}

func TestHCLSchema(t *testing.T) {
	source := []byte("temporary_files = false\ncomponent_store = true\noperation_timeout = \"10m\"\nartifact_paths = [\"C:\\\\Windows\\\\Temp\\\\build.ps1\"]\n")
	file, diags := hclsyntax.ParseConfig(source, "test.pkr.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	p := new(Provisioner)
	value, diags := hcldec.Decode(file.Body, p.ConfigSpec(), nil)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	if value.GetAttr("temporary_files").True() {
		t.Fatal("incorrect bool schema")
	}
	bad, diags := hclsyntax.ParseConfig([]byte("component_store = [true]"), "bad.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	if _, diags = hcldec.Decode(bad.Body, p.ConfigSpec(), nil); !diags.HasErrors() {
		t.Fatal("accepted wrong type")
	}
}

func TestSupportedOperatingSystems(t *testing.T) {
	for _, build := range []int{14393, 17763, 20348, 26100} {
		for _, installation := range []string{"Server", "Server Core"} {
			if !supported(report.Facts{Version: "10.0", Build: build, ProductType: 3, InstallationType: installation, NativeProcess: true, Elevated: true}) {
				t.Fatal(build, installation)
			}
		}
	}
	for _, build := range []int{14393, 19045, 22631, 26100, 26200} {
		if !supported(report.Facts{Version: "10.0", Build: build, ProductType: 1, InstallationType: "Client", NativeProcess: true, Elevated: true}) {
			t.Fatal(build)
		}
	}
	for _, f := range []report.Facts{{Version: "6.3", Build: 9600}, {Version: "10.0", Build: 26100, ProductType: 2, NativeProcess: true, Elevated: true}, {Version: "10.0", Build: 99999, ProductType: 3, InstallationType: "Server", NativeProcess: true, Elevated: true}} {
		if supported(f) {
			t.Fatal("unsupported guest accepted")
		}
	}
}

func TestReportWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "report.json")
	for i := 0; i < 2; i++ {
		if err := writeReport(path, &report.Report{SchemaVersion: 1, Status: "failed", Error: "test"}); err != nil {
			t.Fatal(err)
		}
	}
	if data, err := os.ReadFile(path); err != nil || len(data) == 0 {
		t.Fatal(err)
	}
}
