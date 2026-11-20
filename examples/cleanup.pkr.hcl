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

source "null" "windows" {
  communicator   = "winrm"
  winrm_host     = var.host
  winrm_username = var.username
  winrm_password = var.password
  winrm_use_ssl  = true
  winrm_insecure = false
}

build {
  sources = ["source.null.windows"]
  provisioner "windows-cleanup" {
    temporary_files   = true
    component_store   = true
    dry_run           = true
    operation_timeout = "60m"
    report_path       = "reports/cleanup-dry-run.json"
  }
}
