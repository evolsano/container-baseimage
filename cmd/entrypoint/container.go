package entrypoint

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/osixia/container-baseimage/core"
	"github.com/osixia/container-baseimage/log"
)

var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Container image information",

	Aliases: []string{
		"c",
	},
}

var debugPackages = func() string {
	return strings.Join(core.Instance().Distribution().Config().DebugPackages, "\n")
}

var environmentFilesFunc = func() string {
	efs, err := core.Instance().Filesystem().ListDotEnv()
	if err != nil {
		log.Fatal(err.Error())
	}
	return strings.Join(efs, "\n")
}

var servicesFunc = func() string {
	svcs, err := core.Instance().Services().List()
	if err != nil {
		log.Fatal(err.Error())
	}

	r := ""
	for _, s := range svcs {
		r += s.Status()
	}

	return r
}

func init() {
	// subcommands
	containerCmd.AddCommand(newPrintCmd("debug-packages", "Debug packages", "dbg-pkgs", debugPackages))
	containerCmd.AddCommand(newPrintCmd("environment-files", "Environment file(s)", "env", environmentFilesFunc))
	containerCmd.AddCommand(newPrintCmd("services", "Services", "svcs", servicesFunc))
}
