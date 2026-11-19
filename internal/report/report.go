// Package report defines the versioned host/guest JSON protocol.
package report

import "time"

// Facts describe the guest before or after cleanup. Capabilities are probed,
// rather than inferred from localized product names.
type Facts struct {
	ProductName      string          `json:"product_name"`
	Version          string          `json:"version"`
	Build            int             `json:"build"`
	ProductType      int             `json:"product_type"`
	InstallationType string          `json:"installation_type"`
	Architecture     string          `json:"architecture"`
	Elevated         bool            `json:"elevated"`
	NativeProcess    bool            `json:"native_process"`
	Capabilities     map[string]bool `json:"capabilities"`
	PendingReboot    bool            `json:"pending_reboot"`
	PendingReasons   []string        `json:"pending_reasons"`
	UpdateBusy       bool            `json:"update_busy"`
	WinRM            bool            `json:"winrm"`
}

// Operation distinguishes actual changes from dry-run candidates and skips.
type Operation struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	Message          string   `json:"message,omitempty"`
	Removed          []string `json:"removed"`
	Candidates       []string `json:"candidates"`
	Warnings         []string `json:"warnings"`
	DeletedFileBytes int64    `json:"deleted_file_bytes"`
	FreeBytesBefore  *int64   `json:"free_bytes_before,omitempty"`
	FreeBytesAfter   *int64   `json:"free_bytes_after,omitempty"`
	Output           string   `json:"output,omitempty"`
	DurationSeconds  float64  `json:"duration_seconds"`
}

// Report is written on success and failure, including transport failures.
type Report struct {
	SchemaVersion   int         `json:"schema_version"`
	Profile         string      `json:"profile"`
	DryRun          bool        `json:"dry_run"`
	Started         time.Time   `json:"started"`
	DurationSeconds float64     `json:"duration_seconds"`
	Status          string      `json:"status"`
	Error           string      `json:"error,omitempty"`
	Before          *Facts      `json:"before,omitempty"`
	After           *Facts      `json:"after,omitempty"`
	Operations      []Operation `json:"operations"`
	SysprepStatus   string      `json:"sysprep_status"`
}
