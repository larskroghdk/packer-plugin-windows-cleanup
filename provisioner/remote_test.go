package provisioner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestWorkspaceScript runs the workspace script under Windows PowerShell with
// $env:SystemRoot pointed at a temporary directory inside the session. The
// workspace must admit only SYSTEM and Administrators, and a second run with
// the same ID must refuse the existing directory.
func TestWorkspaceScript(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires Windows PowerShell")
	}
	powershell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("powershell.exe is not available")
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Temp"), 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(script string) (string, error) {
		cmd := exec.Command(powershell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", script)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(kv), "PSMODULEPATH=") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	const id = "00000000-0000-0000-0000-000000000001"
	script := `$env:SystemRoot = ` + quotePS(root) + `; ` + workspaceScript(id)
	dir, err := run(script)
	if err != nil {
		t.Fatalf("first run: %v: %s", err, dir)
	}
	if want := filepath.Join(root, "Temp", "packer-windows-cleanup-"+id); !strings.EqualFold(dir, want) {
		t.Fatalf("workspace = %q, want %q", dir, want)
	}
	quoted := quotePS(dir)
	t.Cleanup(func() {
		if out, err := run(`$a = Get-Acl -LiteralPath ` + quoted + `; $a.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule([System.Security.Principal.WindowsIdentity]::GetCurrent().User, 'FullControl', 'ContainerInherit, ObjectInherit', 'None', 'Allow'))); (New-Object System.IO.DirectoryInfo(` + quoted + `)).SetAccessControl($a)`); err != nil {
			t.Errorf("restore access for cleanup: %v: %s", err, out)
		}
	})
	acl, err := run(`$a = Get-Acl -LiteralPath ` + quoted + `; [string]$a.AreAccessRulesProtected + ' ' + ((@($a.Access | ForEach-Object { $_.IdentityReference.Translate([System.Security.Principal.SecurityIdentifier]).Value + '=' + $_.FileSystemRights }) | Sort-Object) -join ',')`)
	if err != nil {
		t.Fatalf("read ACL: %v: %s", err, acl)
	}
	if want := "True S-1-5-18=FullControl,S-1-5-32-544=FullControl"; acl != want {
		t.Fatalf("ACL = %q, want %q", acl, want)
	}
	if out, err := run(script); err == nil || !strings.Contains(out, "Workspace already exists") {
		t.Fatalf("second run with the same ID = %v: %s, want a refusal", err, out)
	}
}
