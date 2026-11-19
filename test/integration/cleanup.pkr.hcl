packer {
  required_plugins {
    windows-cleanup = {
      source  = "github.com/larskroghdk/windows-cleanup"
      version = ">= 0.1.0"
    }
  }
}
variable "host" { type = string }
variable "username" { type = string }
variable "password" {
  type      = string
  sensitive = true
}
variable "report_path" { type = string }
variable "dry_run" { type = bool }
variable "use_ssl" {
  type    = bool
  default = true
}
source "null" "windows" {
  communicator   = "winrm"
  winrm_host     = var.host
  winrm_username = var.username
  winrm_password = var.password
  winrm_use_ssl  = var.use_ssl
  winrm_insecure = false
}
build {
  sources = ["source.null.windows"]
  provisioner "windows-cleanup" {
    profile                = "custom"
    temporary_files        = false
    provisioning_artifacts = true
    artifact_paths         = ["C:\\ProgramData\\PackerCleanup\\Staging\\integration-owned.txt"]
    validate_system        = true
    dry_run                = var.dry_run
    report_path            = var.report_path
  }
}
