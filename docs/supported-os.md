# Supported operating systems

| Guest | Accepted base builds | Installation |
| --- | --- | --- |
| Windows Server 2016 | 14393 | Server / Server Core |
| Windows Server 2019 | 17763 | Server / Server Core |
| Windows Server 2022 | 20348 | Server / Server Core |
| Windows Server 2025 | 26100 | Server / Server Core |
| Windows 10 | 14393 and later client builds below 22000 | Client |
| Windows 11 | 22000 and later client builds | Client |

Detection reads actual CIM version/build/product type and registry installation type, captures native architecture and product caption, probes tooling, checks elevation/native 64-bit PowerShell, WinRM listeners, CBS/Windows Update reboot keys, pending file renames, pending.xml and pending DISM packages. Windows Update COM busy state and updater/installer processes cause conservative deferral. A service merely being started is not treated as active servicing.

Domain controllers, unknown server base builds, nonnative/32-bit guest processes and pre-1607 client builds are rejected. Client builds use capability gates; future client releases require the VM matrix before claiming validation. Architecture is recorded; AMD64 and ARM64 guests require corresponding native PowerShell, and ARM64 release certification remains a separate matrix entry. Host binaries are independent of guest architecture.

Optional capabilities can be skipped explicitly with `unsupported_policy="skip"`. WER Disk Cleanup and Recycle Bin commands may be unavailable on Core. DISM operations require elevation and no pending servicing. Windows release lifecycle/ESU eligibility is independent of technical plugin acceptance; this plugin does not enable unsupported releases to receive updates.
