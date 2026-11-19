// Command packer-plugin-windows-cleanup cleans Windows template build guests.
package main

import (
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/plugin"
	sdkversion "github.com/hashicorp/packer-plugin-sdk/version"
	"github.com/larskroghdk/packer-plugin-windows-cleanup/provisioner"
	"os"
)

var version = "0.1.0"
var versionPrerelease = "dev"

func main() {
	plugins := plugin.NewSet()
	plugins.RegisterProvisioner(plugin.DEFAULT_NAME, new(provisioner.Provisioner))
	plugins.SetVersion(sdkversion.NewPluginVersion(version, versionPrerelease, ""))
	if err := plugins.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
