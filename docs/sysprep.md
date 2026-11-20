# 🔌 Sysprep and builder shutdown

`sysprep=true` **stages** an administrator-only finalizer; the provisioner never runs Sysprep, shuts down, or treats a communicator disconnect as success. Only the QEMU builder strategy is accepted. Cleanup reports record `staged_for_builder_shutdown`, never successful generalization.

Configure your existing QEMU source with:

```hcl
shutdown_command = "powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File C:\\ProgramData\\PackerCleanup\\Finalization\\Invoke-WindowsCleanupFinalizer.ps1"
shutdown_timeout = "30m"
```

Adjust ProgramData if the guest uses a nondefault volume. Use WinRM, leave Packer's normal shutdown step enabled, and place cleanup last among mutating guest provisioners. Exports can be downloaded before shutdown. Do not add provisioners after generalization, disable the builder shutdown step, or issue a second shutdown command. The provisioner cannot inspect or enforce surrounding HCL; review it before a release build.

The QEMU shutdown step executes its shutdown command and then waits for VM power-off; timeout fails rather than capturing the running guest. The finalizer checks reboot/update completion, health, WinRM, domain state, setup image state and unprovisioned user Appx packages, creates an exclusive attempt marker, launches `sysprep.exe /generalize /oobe /quiet /shutdown` asynchronously, and returns while connectivity still exists. Only the builder waits for shutdown. This avoids the provisioner attempting postflight calls after WinRM has been invalidated.

No second attempt is automatic. Failures retain `%SystemRoot%\System32\Sysprep\Panther` diagnostics and the attempt marker. If Sysprep exits without shutdown, QEMU times out. A successful power-off alone is not proof of successful generalization; verify image setup state and first boot on a clone as a release gate. The finalizer's Appx check is deliberately conservative; remediate provisioning before sealing rather than mass-removing applications. It does not promise to detect every Microsoft Sysprep restriction, upgraded image history or third-party role limitation.

No cloud-specific AWS/Azure deprovisioning is implemented. Their agents may require distinct steps; do not substitute this generic finalizer for the platform's supported image preparation workflow. ResetBase and Sysprep are independently explicit and cannot be rolled back.

The lifecycle follows the [QEMU shutdown implementation](https://github.com/hashicorp/packer-plugin-qemu/blob/main/builder/qemu/step_shutdown.go) and [Microsoft generalization guidance](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/sysprep--generalize--a-windows-installation). Live Sysprep shutdown/clone validation remains mandatory before production use.
