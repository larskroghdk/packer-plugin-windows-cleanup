//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package provisioner

import (
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"regexp"
	"strings"
	"time"
)

// Config is the strongly typed Packer interface. Destructive categories default
// to off; ordinary temporary cleanup and system validation default to on.
type Config struct {
	common.PackerConfig   `mapstructure:",squash"`
	Profile               string         `mapstructure:"profile"`
	TemporaryFiles        config.Trilean `mapstructure:"temporary_files"`
	ComponentStore        bool           `mapstructure:"component_store"`
	WindowsUpdate         bool           `mapstructure:"windows_update"`
	InstallerCleanup      bool           `mapstructure:"installer_cleanup"`
	ProvisioningArtifacts bool           `mapstructure:"provisioning_artifacts"`
	LogCleanup            bool           `mapstructure:"log_cleanup"`
	ErrorReporting        bool           `mapstructure:"error_reporting"`
	EmptyRecycleBin       bool           `mapstructure:"empty_recycle_bin"`
	RemoveUserProfiles    bool           `mapstructure:"remove_user_profiles"`
	ClearEventLogs        bool           `mapstructure:"clear_event_logs"`
	ResetComponentBase    bool           `mapstructure:"reset_component_base"`
	ValidateSystem        config.Trilean `mapstructure:"validate_system"`
	ReportCleanup         config.Trilean `mapstructure:"report_cleanup"`
	DryRun                bool           `mapstructure:"dry_run"`
	FailOnError           config.Trilean `mapstructure:"fail_on_error"`
	UnsupportedPolicy     string         `mapstructure:"unsupported_policy"`
	PendingRebootPolicy   string         `mapstructure:"pending_reboot_policy"`
	LogLevel              string         `mapstructure:"log_level"`
	OperationTimeout      time.Duration  `mapstructure:"operation_timeout"`
	ReportPath            string         `mapstructure:"report_path"`
	CleanupPaths          []string       `mapstructure:"cleanup_paths"`
	InstallerPaths        []string       `mapstructure:"installer_paths"`
	ArtifactPaths         []string       `mapstructure:"artifact_paths"`
	LogPaths              []string       `mapstructure:"log_paths"`
	Exclusions            []string       `mapstructure:"exclusions"`
	BuildProfileSIDs      []string       `mapstructure:"build_profile_sids"`
	EventLogs             []string       `mapstructure:"event_logs"`
	LogExportPath         string         `mapstructure:"log_export_path"`
	MinimumAgeHours       int            `mapstructure:"minimum_age_hours"`
	RequiredServices      []string       `mapstructure:"required_services"`
	RequiredSoftwarePaths []string       `mapstructure:"required_software_paths"`
	RequiredFeatures      []string       `mapstructure:"required_features"`
	Sysprep               bool           `mapstructure:"sysprep"`
	ctx                   interpolate.Context
}

var (
	safePath   = regexp.MustCompile(`(?i)^[a-z]:\\[^<>:"/|?*\x00-\x1f]+$`)
	deviceName = regexp.MustCompile(`^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])$`)
	localSID   = regexp.MustCompile(`^S-1-5-21-(\d+-){3}\d+$`)
)

// validatePath rejects alternate streams, device/UNC paths and ambiguous Win32
// normalization. The guest additionally enforces roots and rejects reparse points.
func validatePath(path string) error {
	if !safePath.MatchString(path) {
		return fmt.Errorf("invalid absolute Windows path %q", path)
	}
	for _, part := range strings.Split(path[3:], `\`) {
		if part == "" || part == "." || part == ".." || strings.TrimRight(part, " .") != part {
			return fmt.Errorf("ambiguous path %q", path)
		}
		stem := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if deviceName.MatchString(stem) {
			return fmt.Errorf("device name in %q", path)
		}
	}
	return nil
}

func (c *Config) prepare(raws ...any) error {
	if err := config.Decode(c, &config.DecodeOpts{PluginType: "windows-cleanup", Interpolate: true, InterpolateContext: &c.ctx}, raws...); err != nil {
		return err
	}
	if c.Profile == "" {
		c.Profile = "safe"
	}
	if c.Profile != "safe" && c.Profile != "custom" {
		return fmt.Errorf("profile must be safe or custom")
	}
	if c.OperationTimeout == 0 {
		c.OperationTimeout = 60 * time.Minute
	}
	if c.OperationTimeout < time.Second || c.OperationTimeout > 24*time.Hour {
		return fmt.Errorf("operation_timeout must be between 1s and 24h")
	}
	if c.MinimumAgeHours == 0 {
		c.MinimumAgeHours = 24
	}
	if c.MinimumAgeHours < 1 {
		return fmt.Errorf("minimum_age_hours must be positive")
	}
	if c.UnsupportedPolicy == "" {
		c.UnsupportedPolicy = "error"
	}
	if c.PendingRebootPolicy == "" {
		c.PendingRebootPolicy = "error"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.UnsupportedPolicy != "error" && c.UnsupportedPolicy != "skip" {
		return fmt.Errorf("unsupported_policy must be error or skip")
	}
	if c.PendingRebootPolicy != "error" && c.PendingRebootPolicy != "skip_servicing" {
		return fmt.Errorf("pending_reboot_policy must be error or skip_servicing")
	}
	if c.LogLevel != "info" && c.LogLevel != "quiet" {
		return fmt.Errorf("log_level must be info or quiet")
	}
	if c.ReportPath == "" {
		c.ReportPath = "windows-cleanup-report.json"
	}
	if c.ResetComponentBase && !c.ComponentStore && !c.WindowsUpdate {
		return fmt.Errorf("reset_component_base requires component_store or windows_update")
	}
	if c.RemoveUserProfiles && len(c.BuildProfileSIDs) == 0 {
		return fmt.Errorf("remove_user_profiles requires build_profile_sids")
	}
	for _, sid := range c.BuildProfileSIDs {
		if !localSID.MatchString(sid) {
			return fmt.Errorf("invalid build profile SID")
		}
	}
	if c.ClearEventLogs && (len(c.EventLogs) == 0 || c.LogExportPath == "") {
		return fmt.Errorf("clear_event_logs requires event_logs and log_export_path")
	}
	for _, name := range c.EventLogs {
		if name == "" || strings.EqualFold(name, "Security") || strings.ContainsAny(name, `\:*?"<>|`) || strings.Contains(name, "..") {
			return fmt.Errorf("invalid or protected event log %q", name)
		}
	}
	for _, paths := range [][]string{c.CleanupPaths, c.InstallerPaths, c.ArtifactPaths, c.LogPaths, c.Exclusions, c.RequiredSoftwarePaths} {
		for _, path := range paths {
			if err := validatePath(path); err != nil {
				return err
			}
		}
	}
	if c.LogExportPath != "" {
		if err := validatePath(c.LogExportPath); err != nil {
			return err
		}
	}
	if c.Sysprep && (c.PackerBuilderType != "qemu" || c.ValidateSystem.False() || c.PendingRebootPolicy != "error") {
		return fmt.Errorf("sysprep stages a builder-owned finalizer: requires qemu, validate_system=true and pending_reboot_policy=error; configure the documented shutdown_command")
	}
	return nil
}

func (c *Config) operations() []string {
	out := []string{}
	if !c.TemporaryFiles.False() && (c.Profile == "safe" || c.TemporaryFiles.True()) {
		out = append(out, "temporary_files")
	}
	for _, op := range []struct {
		id      string
		enabled bool
	}{
		{"installer_cleanup", c.InstallerCleanup}, {"log_cleanup", c.LogCleanup}, {"provisioning_artifacts", c.ProvisioningArtifacts},
		{"error_reporting", c.ErrorReporting}, {"empty_recycle_bin", c.EmptyRecycleBin}, {"remove_user_profiles", c.RemoveUserProfiles},
		{"clear_event_logs", c.ClearEventLogs}, {"component_store", c.ComponentStore || c.WindowsUpdate},
	} {
		if op.enabled {
			out = append(out, op.id)
		}
	}
	if !c.ValidateSystem.False() {
		out = append(out, "validate_system")
	}
	if c.Sysprep {
		out = append(out, "stage_sysprep")
	}
	return out
}

// guestOptions sends data as JSON, never as interpolated shell commands.
func (c *Config) guestOptions() map[string]any {
	return map[string]any{"dry_run": c.DryRun, "reset_component_base": c.ResetComponentBase, "minimum_age_hours": c.MinimumAgeHours,
		"cleanup_paths": c.CleanupPaths, "installer_paths": c.InstallerPaths, "artifact_paths": c.ArtifactPaths, "log_paths": c.LogPaths,
		"exclusions": c.Exclusions, "build_profile_sids": c.BuildProfileSIDs, "event_logs": c.EventLogs, "log_export_path": c.LogExportPath,
		"required_services": c.RequiredServices, "required_software_paths": c.RequiredSoftwarePaths, "required_features": c.RequiredFeatures}
}
