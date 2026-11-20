# 🧹 packer-plugin-windows-cleanup

A Go Packer provisioner for the final cleanup and validation of Windows VM templates over WinRM. Uses the Packer Plugin SDK and embedded Windows PowerShell 5.1; works without a desktop on Server Core.

## 🚦 Release status

**Validation:** implementation and automated local checks are provided. A release must pass the [disposable VM matrix](docs/testing.md) before it is described as production validated. No live image build has been certified by the unit tests.

## 🚀 Quick start

```hcl
packer {
  required_plugins {
    windows-cleanup = {
      source  = "github.com/larskroghdk/windows-cleanup"
      version = ">= 0.1.0"
    }
  }
}

# Inside your existing build, after installation, updates, hardening and restarts:
build {
  sources = ["source.qemu.windows"]
  provisioner "windows-cleanup" {
    temporary_files        = true
    component_store        = true
    windows_update         = true
    installer_cleanup      = true
    provisioning_artifacts = true
    installer_paths        = ["C:\\ProgramData\\PackerCleanup\\Staging\\installers"]
    artifact_paths         = ["C:\\Windows\\Temp\\build-secrets.json"]
    dry_run                = true
    report_path            = "reports/windows-cleanup.json"
  }
}
```

Remove `dry_run` after reviewing a disposable VM report. `windows_update` selects the same DISM component cleanup as `component_store`; enabling both runs it once. No update datastore, Installer cache, WinSxS contents, activation state, firewall rule or WinRM configuration is deleted or changed.

## 🛠️ Build and install

Build and install locally before a release exists:

```powershell
go build -ldflags "-X main.versionPrerelease=" -o packer-plugin-windows-cleanup.exe .
packer plugins install --path ./packer-plugin-windows-cleanup.exe github.com/larskroghdk/windows-cleanup
```

## 🛡️ Defaults and safety

The plugin defaults to stale temporary file cleanup (24 hours), system validation, failure on errors, and a host JSON report. Servicing, profile removal, WER cleanup, event log clearing, Recycle Bin cleanup, ResetBase and Sysprep staging are explicit opt-ins. A pending reboot or active update stops the build by default. Use Packer's `windows-restart` before this provisioner.

## 📚 Documentation

| Guide | What you will find |
| --- | --- |
| ⚙️ [Configuration](docs/configuration.md) | Options, defaults and allowed paths |
| 🏗️ [Architecture](docs/architecture.md) | SDK lifecycle, guest protocol and design decisions |
| 🖥️ [Supported systems](docs/supported-os.md) | Operating systems and capability boundaries |
| 🛡️ [Safety and reporting](docs/safety.md) | Protected resources, cleanup boundaries and JSON reports |
| 🔌 [Sysprep lifecycle](docs/sysprep.md) | Finalizer staging and builder-owned shutdown |
| 🧪 [Testing](docs/testing.md) | Local checks, disposable VM tests and release gates |
| 📋 [Validation record](docs/validation.md) | Recorded checks and outstanding validation |
| 🗺️ [Legacy operation mapping](docs/legacy-mapping.md) | Legacy operations and their disposition |
| 🗃️ [Legacy source inventory](docs/legacy-inventory.md) | Reviewed source files |
| 📖 [Research sources](docs/research.md) | References supporting implementation decisions |
