## Summary

<!-- What changes and why? -->

- [ ] User-visible changes are described in the commit message; release notes are generated from commits, or N/A

## Related issues

<!-- `Closes #` / `Refs #`. New cleanup operations should start with an issue. -->

## Compatibility

- [ ] No breaking change, or the summary names the changed option, default or behavior and the migration
- [ ] Report JSON change increments `schema_version` and updates [Safety and reports](https://github.com/larskroghdk/packer-plugin-windows-cleanup/blob/main/docs/safety.md), or N/A
- [ ] `Config` change is regenerated with `go generate ./provisioner` and documented in [Configuration](https://github.com/larskroghdk/packer-plugin-windows-cleanup/blob/main/docs/configuration.md), or N/A
- [ ] Tested and untested supported Windows versions and installation types (Server Core, client) are listed

## Testing

- [ ] `./scripts/test.ps1` passes
- [ ] `go generate ./provisioner` leaves no diff
- [ ] Changed cleanup logic has behavior tests for dry run, path boundaries, reparse points, exclusions and failures where they apply, or N/A
- [ ] A second run on the same guest is idempotent (no new removals and no failure), or N/A
- [ ] Error handling covers native exit codes and `fail_on_error` semantics where relevant, or N/A
- [ ] Guest tests stay inside Pester `TestDrive`
- [ ] Live disposable VM run: builder, guest OS/build and report attached, or "Not run" with reason
- [ ] After a live run, `validate_system` passes, `DISM /Online /Cleanup-Image /CheckHealth` reports no corruption, WinRM postflight succeeds, and no pending reboot was introduced, or "Not run" with reason

Mocks and local checks do not validate a real Windows image. Report them separately from live results.

## Cleanup safety

- [ ] Deletion roots are not widened; exclusions and manifest restrictions are preserved, or N/A
- [ ] Windows Installer, WinSxS, servicing stores, production profiles, networking, activation and WinRM are preserved, or N/A
- [ ] New destructive operation has explicit configuration, a source rationale in [legacy mapping](https://github.com/larskroghdk/packer-plugin-windows-cleanup/blob/main/docs/legacy-mapping.md), dry-run behavior and reporting, or N/A
- [ ] Guest calls go through `remote.action`; user data stays out of command text
- [ ] Transport, timeout and final-validation failures still fail the build; reboot and servicing gates are kept
- [ ] The provisioner still never shuts down or generalizes the guest
- [ ] The change does not disrupt the Packer session, WinRM, Windows servicing or builder-owned Sysprep finalizer staging
- [ ] No new registry writes, or they are justified, restored and reported
- [ ] No credentials or secrets in code, reports, logs or artifacts
- [ ] Actions remain pinned to commit SHAs with minimum permissions, or N/A

Security impact:

## Documentation

- [ ] README, docs/, examples and AGENTS.md updated where behavior changed, or N/A
- [ ] `.ps1` files remain ASCII with CRLF line endings, or N/A
