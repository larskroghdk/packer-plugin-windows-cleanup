package provisioner

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/larskroghdk/packer-plugin-windows-cleanup/internal/report"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

type fakeCommunicator struct {
	files      map[string][]byte
	facts      report.Facts
	operations []string
	options    []map[string]any
	fail       string
	disconnect bool
	wait       bool
	malformed  bool
}

func newFake() *fakeCommunicator {
	return &fakeCommunicator{files: map[string][]byte{}, facts: report.Facts{ProductName: "Server", Version: "10.0.20348", Build: 20348, ProductType: 3, InstallationType: "Server Core", Architecture: "AMD64", NativeProcess: true, Elevated: true, WinRM: true, Capabilities: map[string]bool{"dism": true, "wer_cleanup": true, "recycle_bin": true}}}
}
func decodeCommand(command string) string {
	b, _ := base64.StdEncoding.DecodeString(strings.Fields(command)[len(strings.Fields(command))-1])
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(units))
}

var actionPattern = regexp.MustCompile(`-Action '([^']+)' -InputPath '([^']+)' -OutputPath '([^']+)'`)

func (f *fakeCommunicator) Start(ctx context.Context, cmd *packer.RemoteCmd) error {
	script := decodeCommand(cmd.Command)
	if strings.Contains(script, "New-Item -ItemType Directory -Path $d") {
		fmt.Fprintln(cmd.Stdout, `C:\Windows\Temp\packer-windows-cleanup-1234`)
		cmd.SetExited(0)
		return nil
	}
	match := actionPattern.FindStringSubmatch(script)
	if len(match) == 0 {
		cmd.SetExited(0)
		return nil
	}
	if f.wait {
		go func() { <-ctx.Done(); cmd.SetExited(1) }()
		return nil
	}
	if f.disconnect {
		return fmt.Errorf("unexpected WinRM disconnect")
	}
	var value any
	if match[1] == "Detect" {
		value = f.facts
	} else {
		var options map[string]any
		if err := json.Unmarshal(f.files[match[2]], &options); err != nil {
			return err
		}
		id := options["operation"].(string)
		f.operations = append(f.operations, id)
		f.options = append(f.options, options)
		op := report.Operation{ID: id, Status: "success", Removed: []string{}, Candidates: []string{}, Warnings: []string{}}
		if options["dry_run"].(bool) {
			op.Status = "dry_run"
		}
		if id == f.fail {
			op.Status = "failed"
			op.Message = "injected failure"
		}
		value = op
	}
	data, _ := json.Marshal(value)
	if f.malformed {
		data = []byte("broken")
	}
	f.files[match[3]] = data
	cmd.SetExited(0)
	return nil
}
func (f *fakeCommunicator) Upload(path string, r io.Reader, _ *os.FileInfo) error {
	b, e := io.ReadAll(r)
	f.files[path] = b
	return e
}
func (f *fakeCommunicator) Download(path string, w io.Writer) error {
	b, ok := f.files[path]
	if !ok {
		return fmt.Errorf("missing %s", path)
	}
	_, e := w.Write(b)
	return e
}
func (*fakeCommunicator) UploadDir(string, string, []string) error {
	return fmt.Errorf("unexpected UploadDir")
}
func (*fakeCommunicator) DownloadDir(string, string, []string) error {
	return fmt.Errorf("unexpected DownloadDir")
}

func runFake(t *testing.T, f *fakeCommunicator, raw map[string]any) (*report.Report, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	raw["report_path"] = path
	var p Provisioner
	if err := p.Prepare(raw); err != nil {
		t.Fatal(err)
	}
	ui := &packer.BasicUi{Reader: strings.NewReader(""), Writer: io.Discard, ErrorWriter: io.Discard}
	err := p.Provision(context.Background(), ui, f, map[string]any{"ConnType": "winrm"})
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var r report.Report
	if e := json.Unmarshal(data, &r); e != nil {
		t.Fatal(e)
	}
	return &r, err
}

func TestProvisionLifecycle(t *testing.T) {
	f := newFake()
	r, err := runFake(t, f, map[string]any{"dry_run": true, "component_store": true, "windows_update": true, "artifact_paths": []string{`C:\Windows\Temp\secret.json`}, "provisioning_artifacts": true, "exclusions": []string{`C:\Windows\Temp\keep`}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "success" || r.Before == nil || r.After == nil {
		t.Fatalf("%+v", r)
	}
	if len(f.operations) != 4 {
		t.Fatal(f.operations)
	}
	for _, o := range f.options {
		if !o["dry_run"].(bool) {
			t.Fatal("dry run lost")
		}
	}
}
func TestProvisionFailurePolicies(t *testing.T) {
	for _, keepGoing := range []bool{false, true} {
		f := newFake()
		f.fail = "temporary_files"
		r, err := runFake(t, f, map[string]any{"fail_on_error": !keepGoing})
		if (err == nil) != keepGoing {
			t.Fatal(err)
		}
		if keepGoing && r.Status != "warnings" {
			t.Fatal(r.Status)
		}
	}
	f := newFake()
	f.facts.PendingReboot = true
	if _, err := runFake(t, f, map[string]any{}); err == nil || len(f.operations) != 0 {
		t.Fatal("pending reboot must block changes")
	}
	f = newFake()
	f.facts.PendingReboot = true
	r, err := runFake(t, f, map[string]any{"pending_reboot_policy": "skip_servicing", "component_store": true})
	if err != nil || r.Status != "warnings" {
		t.Fatal(err, r.Status)
	}
	f = newFake()
	f.facts.Capabilities["wer_cleanup"] = false
	r, err = runFake(t, f, map[string]any{"error_reporting": true, "unsupported_policy": "skip"})
	if err != nil || r.Status != "warnings" {
		t.Fatal(err, r.Status)
	}
	f = newFake()
	f.facts.WinRM = false
	if _, err := runFake(t, f, map[string]any{}); err == nil {
		t.Fatal("WinRM failure accepted")
	}
}
func TestTransportAndTimeout(t *testing.T) {
	for _, mode := range []string{"disconnect", "malformed", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			f := newFake()
			f.disconnect = mode == "disconnect"
			f.malformed = mode == "malformed"
			f.wait = mode == "timeout"
			start := time.Now()
			r, err := runFake(t, f, map[string]any{"operation_timeout": "1s"})
			if err == nil || r.Status != "failed" {
				t.Fatal("failure accepted")
			}
			if time.Since(start) > 5*time.Second {
				t.Fatal("timeout not bounded")
			}
		})
	}
}
func TestSysprepIsOnlyStaged(t *testing.T) {
	f := newFake()
	r, err := runFake(t, f, map[string]any{"sysprep": true, "packer_builder_type": "qemu"})
	if err != nil {
		t.Fatal(err)
	}
	if r.SysprepStatus != "staged_for_builder_shutdown" || f.operations[len(f.operations)-1] != "stage_sysprep" {
		t.Fatal(r.SysprepStatus, f.operations)
	}
	f = newFake()
	f.fail = "temporary_files"
	r, err = runFake(t, f, map[string]any{"sysprep": true, "packer_builder_type": "qemu", "fail_on_error": false})
	if err != nil {
		t.Fatal(err)
	}
	if r.SysprepStatus != "blocked" {
		t.Fatal("staged after failure")
	}
}

func TestActiveUpdatesAndArtifactFailuresCannotBeIgnored(t *testing.T) {
	f := newFake()
	f.facts.UpdateBusy = true
	if _, err := runFake(t, f, map[string]any{"pending_reboot_policy": "skip_servicing", "fail_on_error": false}); err == nil || len(f.operations) != 0 {
		t.Fatal("active updating must block every cleanup category")
	}
	f = newFake()
	f.fail = "provisioning_artifacts"
	if _, err := runFake(t, f, map[string]any{"provisioning_artifacts": true, "fail_on_error": false}); err == nil {
		t.Fatal("explicit secret-bearing artifact removal cannot be ignored")
	}
	f = newFake()
	r, err := runFake(t, f, map[string]any{"sysprep": true, "packer_builder_type": "qemu", "dry_run": true})
	if err != nil || r.SysprepStatus != "dry_run" {
		t.Fatal(err, r.SysprepStatus)
	}
}
