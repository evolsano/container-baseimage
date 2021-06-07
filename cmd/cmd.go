package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/osixia/container-baseimage/cmd/entrypoint"
	"github.com/osixia/container-baseimage/cmd/logger"
	"github.com/osixia/container-baseimage/cmd/processes"
	"github.com/osixia/container-baseimage/cmd/services"
)

const (
	installGroupID    = "install"
	entrypointGroupID = "entrypoint"
	containerGroupID  = "container"
	envsubstGroupID   = "envsubst"
	loggerGroupID     = "logger"
)

var cmd = &cobra.Command{
	Use: "container-baseimage",

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logger.HandleFlags(cmd)
	},
}

func init() {
	cobra.EnableCommandSorting = false

	// subcommands groups
	cmd.AddGroup(&cobra.Group{
		ID:    installGroupID,
		Title: "Install Command:",
	})

	cmd.AddGroup(&cobra.Group{
		ID:    entrypointGroupID,
		Title: "Entrypoint Command:",
	})

	cmd.AddGroup(&cobra.Group{
		ID:    containerGroupID,
		Title: "Container Commands:",
	})

	cmd.AddGroup(&cobra.Group{
		ID:    envsubstGroupID,
		Title: "Envsubst Commands:",
	})

	cmd.AddGroup(&cobra.Group{
		ID:    loggerGroupID,
		Title: "Logger Command:",
	})

	// subcommands
	entrypoint.EntrypointCmd.GroupID = entrypointGroupID
	cmd.AddCommand(entrypoint.EntrypointCmd)

	services.ServicesCmd.GroupID = containerGroupID
	processes.ProcessesCmd.GroupID = containerGroupID
	cmd.AddCommand(services.ServicesCmd)
	cmd.AddCommand(processes.ProcessesCmd)

	cmd.AddCommand(installCmd)

	cmd.AddCommand(envsubstCmd)
	cmd.AddCommand(envsubstTemplatesCmd)

	logger.LoggerCmd.GroupID = loggerGroupID
	cmd.AddCommand(logger.LoggerCmd)
}

func Run(ctx context.Context) error {
	return cmd.ExecuteContext(ctx)
}
