# 🏗️ Architecture and reference review

The source repository `../packer-plugin-windows-optimize` was reviewed as the architectural foundation: `main.go`, SDK registration, Config/FlatConfig HCL2 schema, communicator execution, facts and release classification, planner and journal contracts, embedded scripts, Go and Pester tests, tagged live tests, documentation, Makefile, GitHub build workflow, GoReleaser configuration, AGENTS/INVARIANTS and its local development skills.

## 🧩 Reused patterns

| Reference pattern | Cleanup implementation and assessment |
| --- | --- |
| `plugin.NewSet`, default provisioner, SDK version | Same registration and describe protocol; independent module and release version |
| Embedded `common.PackerConfig`, Decode/Trilean, SDK schema generator | Strong booleans/lists/durations; generated `config.hcl2spec.go`; no handwritten parallel schema |
| Embedded PS5.1 scripts and action JSON files | Same deployment contract; JSON carries user data, encoded commands carry only generated paths and fixed action names |
| Unique temporary guest workspace | Uses detected SystemRoot, validates ancestry and applies a protected Administrators/SYSTEM ACL; root is never configurable |
| OS facts and capabilities | CIM/registry detection rather than localized name matching; rejects DCs, nonnative/unelevated sessions and unknown server releases |
| Context-aware RemoteCmd | Commands use operation deadlines; nonzero exit and unexpected disconnect fail. SDK transfers are synchronous and rely on communicator transport timeouts |
| WinRM protection and postflight | Never writes service, auth, listener, network or firewall settings; real postflight command and listener/service checks |
| Planner and rollback journal | Not copied: deleted files and ResetBase cannot be rolled back. Explicit manifests and exclusions replace optimization selectors |
| Restart orchestration | Not copied: build owner must finish updates/restart before cleanup; no competing reboot logic |
| Reports on failure | Host report writes atomically, even for detection/transport failures; operation statuses retain partial work |
| Documentation and tests | Go protocol/config tests, PS5.1 Pester behavior tests, opt-in disposable guest tests; local instructions focus on cleanup boundaries |
| GitHub Actions/GoReleaser | Windows checks, generated-schema check, static analysis, embedded scripts, signed draft releases. Host binaries also target Linux/macOS because the Packer host need not be Windows |

## 🔄 Flow

Prepare decodes, defaults and validates config. Provision creates the private workspace, uploads scripts and input JSON, detects facts, rejects unsafe build state, executes selected categories in a fixed order, validates the system, optionally stages a finalizer, and runs postflight detection. It then removes only its own workspace and writes the host report. A timeout fails the build and leaves the workspace because a servicing child may still be running.

Component cleanup is coalesced with Windows Update cleanup. Fresh facts are obtained after servicing so a new reboot requirement blocks later validation/finalization. `fail_on_error=false` allows ordinary category failures to continue as warnings, but validation, finalizer and transport failures remain fatal. Incomplete earlier operations block Sysprep staging.

The reference's reversible optimization model is a good fit for registry/service tuning, but not for deletion. Cleanup deliberately has no rollback claim, broad catalog abstraction, arbitrary command execution interface, dependency on legacy organization scripts, or automatic repair of a damaged component store.

## 🚦 Limits that affect release readiness

Packer provisioners cannot inspect later provisioner blocks or a builder's shutdown settings. Sysprep staging therefore cannot enforce the surrounding HCL lifecycle. The operator must use the documented QEMU shutdown contract and validate the deployed clone. Other builders/cloud agents require their own tested strategies. Guest OS detection and native servicing still require real VM matrix validation; mocks do not certify those mechanisms.
