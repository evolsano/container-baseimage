package debian

import (
	"embed"

	"github.com/osixia/container-baseimage/core"
)

// list all services so .priority files are included (. files are ignored in subdirs otherwise)

//go:embed assets/* assets/services/cron/* assets/services/logrotate/* assets/services/syslog-ng/*
var assets embed.FS

var SupportedDistribution = &core.SupportedDistribution{
	Name:    "Debian & derivatives",
	Vendors: []string{"debian", "ubuntu"},

	Config: &core.DistributionConfig{
		DebugPackages: []string{"curl", "less", "procps", "psmisc", "strace", "vim-tiny"},
		Assets:        []*embed.FS{&assets},

		InstallScript: "install.sh",

		BinPackagesIndexUpdate:  "packages-index-update",
		BinPackagesInstallClean: "packages-install-clean",
		BinPackagesIndexClean:   "packages-index-clean",
	},
}
