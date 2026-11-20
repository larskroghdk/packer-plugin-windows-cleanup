.PHONY: check test generate build
check:
	pwsh -NoProfile -File scripts/test.ps1
test:
	go test ./...
generate:
	go generate ./provisioner
build:
	go build -o dist/packer-plugin-windows-cleanup.exe .
