package provisioner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/uuid"
	"github.com/larskroghdk/packer-plugin-windows-cleanup/guest"
	"io"
	"strings"
	"unicode/utf16"
)

type remote struct {
	comm packer.Communicator
	dir  string
	seq  int
}

func encodedCommand(script string) string {
	units := utf16.Encode([]rune(script))
	data := make([]byte, 2*len(units))
	for i, v := range units {
		binary.LittleEndian.PutUint16(data[2*i:], v)
	}
	return "powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand " + base64.StdEncoding.EncodeToString(data)
}

func (r *remote) exec(ctx context.Context, script string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	cmd := &packer.RemoteCmd{Command: encodedCommand(script), Stdout: &stdout, Stderr: &stderr}
	quietUI := &packer.BasicUi{Reader: strings.NewReader(""), Writer: io.Discard, ErrorWriter: io.Discard}
	if err := cmd.RunWithUi(ctx, r.comm, quietUI); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cmd.ExitStatus() != 0 {
		return nil, fmt.Errorf("guest command exited with status %d (diagnostics retained in guest workspace)", cmd.ExitStatus())
	}
	return stdout.Bytes(), nil
}

func (r *remote) install(ctx context.Context) error {
	// A fresh directory with a protected ACL prevents unprivileged path replacement.
	script := `$ErrorActionPreference='Stop'; $base=Join-Path $env:SystemRoot 'Temp'; $p=$base; while($p){if((Get-Item -LiteralPath $p -Force).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Reparse point in workspace ancestry'};$p=Split-Path -Parent $p}; $d=Join-Path $base 'packer-windows-cleanup-` + uuid.TimeOrderedUUID() + `'; $null=New-Item -ItemType Directory -Path $d; $acl=New-Object Security.AccessControl.DirectorySecurity; $acl.SetAccessRuleProtection($true,$false); foreach($sid in @('S-1-5-18','S-1-5-32-544')){$id=New-Object Security.Principal.SecurityIdentifier($sid);$rule=New-Object Security.AccessControl.FileSystemAccessRule($id,'FullControl','ContainerInherit,ObjectInherit','None','Allow');$acl.AddAccessRule($rule)}; Set-Acl -LiteralPath $d -AclObject $acl; Write-Output $d`
	data, err := r.exec(ctx, script)
	if err != nil {
		return fmt.Errorf("create private workspace: %w", err)
	}
	r.dir = strings.TrimSpace(string(data))
	if err := validatePath(r.dir); err != nil {
		return fmt.Errorf("invalid workspace returned by guest: %w", err)
	}
	scripts, err := guest.Scripts()
	if err != nil {
		return err
	}
	for name, data := range scripts {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.comm.Upload(r.dir+`\`+name, bytes.NewReader(data), nil); err != nil {
			return fmt.Errorf("upload guest scripts: %w", err)
		}
	}
	return nil
}

func quotePS(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func (r *remote) action(ctx context.Context, action string, input any, result any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.seq++
	prefix := fmt.Sprintf(`%s\%03d`, r.dir, r.seq)
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	if err := r.comm.Upload(prefix+"-input.json", bytes.NewReader(data), nil); err != nil {
		return fmt.Errorf("upload action input: %w", err)
	}
	script := `$ErrorActionPreference='Stop'; & ` + quotePS(r.dir+`\`+guest.EntryScript) + ` -Action ` + quotePS(action) + ` -InputPath ` + quotePS(prefix+"-input.json") + ` -OutputPath ` + quotePS(prefix+"-output.json")
	if _, err := r.exec(ctx, script); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var out bytes.Buffer
	if err := r.comm.Download(prefix+"-output.json", &out); err != nil {
		return fmt.Errorf("download action result: %w", err)
	}
	if err := json.Unmarshal(bytes.TrimPrefix(out.Bytes(), []byte{0xef, 0xbb, 0xbf}), result); err != nil {
		return fmt.Errorf("decode %s result: %w", action, err)
	}
	return nil
}

func (r *remote) cleanup(ctx context.Context) error {
	if r.dir == "" {
		return nil
	}
	// Delete only the generated workspace, refusing any link below it.
	_, err := r.exec(ctx, `$ErrorActionPreference='Stop'; $d=`+quotePS(r.dir)+`; if((Split-Path -Leaf $d) -notmatch '^packer-windows-cleanup-[0-9a-f-]+$'){throw 'Invalid workspace'}; if(Test-Path -LiteralPath $d){$p=$d;while($p){if((Get-Item -LiteralPath $p -Force).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Linked workspace'};$p=Split-Path -Parent $p}; function Remove-WorkspaceItem($p){$i=Get-Item -LiteralPath $p -Force;if($i.Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Linked workspace item'};if($i.PSIsContainer){foreach($c in Get-ChildItem -LiteralPath $p -Force){Remove-WorkspaceItem $c.FullName}};Remove-Item -LiteralPath $p -Force};Remove-WorkspaceItem $d}`)
	return err
}
