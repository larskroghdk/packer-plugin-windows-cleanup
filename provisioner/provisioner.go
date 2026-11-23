// Package provisioner implements the Packer cleanup lifecycle over WinRM.
package provisioner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/larskroghdk/packer-plugin-windows-cleanup/internal/report"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Provisioner runs Windows cleanup through the builder's WinRM communicator.
type Provisioner struct{ config Config }

var _ packer.Provisioner = (*Provisioner)(nil)

// ConfigSpec returns the SDK-generated HCL2 schema.
func (p *Provisioner) ConfigSpec() hcldec.ObjectSpec { return p.config.FlatMapstructure().HCL2Spec() }

// Prepare decodes configuration and rejects unsafe or inconsistent options.
func (p *Provisioner) Prepare(raws ...any) error { return p.config.prepare(raws...) }

func supported(f report.Facts) bool {
	if !strings.HasPrefix(f.Version, "10.") || f.Build < 14393 || !f.NativeProcess || !f.Elevated {
		return false
	}
	if f.ProductType == 1 {
		return strings.EqualFold(f.InstallationType, "Client")
	}
	if f.ProductType != 3 {
		return false
	}
	switch f.Build {
	case 14393, 17763, 20348, 26100:
	default:
		return false
	}
	return strings.EqualFold(f.InstallationType, "Server") || strings.EqualFold(f.InstallationType, "Server Core")
}

func writeReport(path string, r *report.Report) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".cleanup-report-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

// Provision detects, cleans and validates the guest, then persists the report.
// Shutdown and generalization remain the builder's responsibility.
func (p *Provisioner) Provision(ctx context.Context, ui packer.Ui, comm packer.Communicator, generatedData map[string]any) (runErr error) {
	cfg := &p.config
	r := &report.Report{SchemaVersion: 1, Profile: cfg.Profile, DryRun: cfg.DryRun, Started: time.Now().UTC(), Status: "failed", Operations: []report.Operation{}, SysprepStatus: "disabled"}
	remote := &remote{comm: comm}
	defer func() {
		// A canceled native servicing process may still be running. Preserve its
		// workspace and fail the build; never treat a disconnect as success.
		if ctx.Err() == nil && !errors.Is(runErr, context.DeadlineExceeded) {
			cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := remote.cleanup(cleanupCtx); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("workspace cleanup: %w", err))
			}
		}
		r.DurationSeconds = time.Since(r.Started).Seconds()
		if runErr != nil {
			r.Error = runErr.Error()
			r.Status = "failed"
		} else if r.Status != "warnings" {
			r.Status = "success"
		}
		if !cfg.ReportCleanup.False() {
			if err := writeReport(cfg.ReportPath, r); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("write cleanup report: %w", err))
			}
		}
		ui.Say(fmt.Sprintf("Windows cleanup: %s (%d operations)", r.Status, len(r.Operations)))
	}()
	if conn, _ := generatedData["ConnType"].(string); conn != "" && conn != "winrm" {
		return fmt.Errorf("windows-cleanup requires WinRM")
	}
	action := func(name string, input any, result any) error {
		opCtx, cancel := context.WithTimeout(ctx, cfg.OperationTimeout)
		defer cancel()
		return remote.action(opCtx, name, input, result)
	}
	installCtx, cancel := context.WithTimeout(ctx, cfg.OperationTimeout)
	err := remote.install(installCtx)
	cancel()
	if err != nil {
		return err
	}
	var before report.Facts
	if err := action("Detect", nil, &before); err != nil {
		return err
	}
	initial := before
	r.Before = &initial
	if !supported(before) {
		return fmt.Errorf("unsupported Windows guest or unelevated/non-native PowerShell process (build %d, type %d)", before.Build, before.ProductType)
	}
	if !before.WinRM {
		return fmt.Errorf("WinRM service/listener validation failed before cleanup")
	}
	if before.UpdateBusy {
		return fmt.Errorf("active update or installer detected; wait for completion before cleanup")
	}
	if before.PendingReboot && cfg.PendingRebootPolicy == "error" {
		return fmt.Errorf("pending reboot detected; restart with windows-restart before cleanup, or set pending_reboot_policy to skip_servicing")
	}
	blocked := false
	for _, id := range cfg.operations() {
		op := report.Operation{ID: id, Removed: []string{}, Candidates: []string{}, Warnings: []string{}}
		capability := ""
		switch id {
		case "component_store", "validate_system":
			capability = "dism"
		case "empty_recycle_bin":
			capability = "recycle_bin"
		case "error_reporting":
			capability = "wer_cleanup"
		}
		if id == "stage_sysprep" && blocked {
			op.Status = "skipped"
			op.Message = "earlier cleanup or validation was incomplete"
			r.SysprepStatus = "blocked"
		} else if before.PendingReboot && (id == "component_store" || id == "validate_system" || id == "stage_sysprep") {
			op.Status = "skipped"
			op.Message = "pending reboot; servicing deferred"
			blocked = true
			r.Status = "warnings"
		} else if capability != "" && !before.Capabilities[capability] {
			op.Status = "skipped"
			op.Message = "required capability unavailable: " + capability
			blocked = true
			r.Status = "warnings"
			if cfg.UnsupportedPolicy == "error" {
				op.Status = "failed"
			}
		} else {
			if cfg.LogLevel != "quiet" {
				ui.Say("Windows cleanup: " + id)
			}
			input := cfg.guestOptions()
			input["operation"] = id
			if err := action("Run", input, &op); err != nil {
				op.ID = id
				op.Status = "failed"
				op.Message = err.Error()
				r.Operations = append(r.Operations, op)
				return err
			}
			if op.ID != id || (op.Status != "success" && op.Status != "skipped" && op.Status != "dry_run" && op.Status != "warning" && op.Status != "failed") {
				return fmt.Errorf("invalid guest result for %s", id)
			}
		}
		r.Operations = append(r.Operations, op)
		if op.Status == "failed" {
			blocked = true
			if !cfg.FailOnError.False() || id == "validate_system" || id == "stage_sysprep" || id == "provisioning_artifacts" {
				return fmt.Errorf("%s: %s", id, op.Message)
			}
			r.Status = "warnings"
		}
		if op.Status == "warning" {
			blocked = true
			r.Status = "warnings"
		}
		if id == "stage_sysprep" && op.Status == "success" {
			r.SysprepStatus = "staged_for_builder_shutdown"
		}
		if id == "stage_sysprep" && op.Status == "dry_run" {
			r.SysprepStatus = "dry_run"
		}
		// Servicing may have introduced a reboot requirement. Recheck before any
		// further servicing/finalization rather than relying on the initial snapshot.
		if id == "component_store" {
			if err := action("Detect", nil, &before); err != nil {
				return err
			}
		}
	}
	var after report.Facts
	if err := action("Detect", nil, &after); err != nil {
		return err
	}
	r.After = &after
	if !after.WinRM {
		return fmt.Errorf("WinRM service/listener validation failed after cleanup")
	}
	if after.UpdateBusy {
		return fmt.Errorf("update or installer became active during cleanup; do not capture this image")
	}
	if after.PendingReboot {
		if cfg.PendingRebootPolicy == "error" {
			return fmt.Errorf("cleanup left pending servicing/reboot; restart before capture")
		}
		r.Status = "warnings"
	}
	return nil
}
