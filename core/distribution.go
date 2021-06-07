package core

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/osixia/container-baseimage/errors"
	"github.com/osixia/container-baseimage/helpers"
	"github.com/osixia/container-baseimage/log"
)

// list generator templates environment and service-name so .env and .priority files are included (. files are ignored in subdirs otherwise)

//go:embed assets/* assets/generator/templates/environment/* assets/generator/templates/services/service-name/*
var assets embed.FS

var coreDistributionConfig = &DistributionConfig{
	Assets: []*embed.FS{&assets},

	BinDest: "/usr/sbin",
}

// Supported distribution
// =============================

type SupportedDistribution struct {
	Name    string
	Vendors []string

	Config *DistributionConfig
}

// Distribution config
// =============================

type DistributionConfig struct {
	DebugPackages []string

	Assets []*embed.FS

	InstallScript string

	BinDest string

	BinPackagesIndexUpdate  string
	BinPackagesInstallClean string
	BinPackagesIndexClean   string
}

func (dc *DistributionConfig) Merge(mdc *DistributionConfig) {

	if mdc.DebugPackages != nil {
		dc.DebugPackages = append(dc.DebugPackages, mdc.DebugPackages...)
	}

	if mdc.Assets != nil {
		dc.Assets = append(dc.Assets, mdc.Assets...)
	}

	if mdc.InstallScript != "" {
		dc.InstallScript = mdc.InstallScript
	}

	if mdc.BinDest != "" {
		dc.BinDest = mdc.BinDest
	}

	if mdc.BinPackagesIndexUpdate != "" {
		dc.BinPackagesIndexUpdate = mdc.BinPackagesIndexUpdate
	}
	if mdc.BinPackagesInstallClean != "" {
		dc.BinPackagesInstallClean = mdc.BinPackagesInstallClean
	}
	if mdc.BinPackagesIndexClean != "" {
		dc.BinPackagesIndexClean = mdc.BinPackagesIndexClean
	}

}

func (dc *DistributionConfig) Validate() (bool, error) {

	if dc.InstallScript == "" {
		return false, fmt.Errorf("InstallScript: %w", errors.ErrRequired)
	}

	if dc.BinDest == "" {
		return false, fmt.Errorf("BinDest: %w", errors.ErrRequired)
	}

	if dc.BinPackagesIndexUpdate == "" {
		return false, fmt.Errorf("BinPackagesIndexUpdate: %w", errors.ErrRequired)
	}
	if dc.BinPackagesInstallClean == "" {
		return false, fmt.Errorf("BinPackagesInstallClean: %w", errors.ErrRequired)
	}
	if dc.BinPackagesIndexClean == "" {
		return false, fmt.Errorf("BinPackagesIndexClean: %w", errors.ErrRequired)
	}

	return true, nil
}

// Distribution
// =============================

type Distribution interface {
	Name() string
	Vendor() string
	Version() string
	VersionCodename() string

	InstallPackages(ctx context.Context, packages []string) error

	PackagesIndexUpdate(ctx context.Context) error
	PackagesIndexClean(ctx context.Context) error
	PackagesInstallClean(ctx context.Context, packages []string) error

	Config() *DistributionConfig
}

type distribution struct {
	name            string
	vendor          string
	version         string
	versionCodename string // can be empty

	config *DistributionConfig
}

func newDistribution(sds []*SupportedDistribution) (Distribution, error) {

	f, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dist := &distribution{}

	s := bufio.NewScanner(f)
	for s.Scan() {
		if m := regexp.MustCompile(`^PRETTY_NAME=(.*)$`).FindStringSubmatch(s.Text()); m != nil {
			dist.name = strings.Trim(m[1], `"`)
		} else if m := regexp.MustCompile(`^ID=(.*)$`).FindStringSubmatch(s.Text()); m != nil {
			dist.vendor = strings.Trim(m[1], `"`)
		} else if m := regexp.MustCompile(`^VERSION_ID=(.*)$`).FindStringSubmatch(s.Text()); m != nil {
			dist.version = strings.Trim(m[1], `"`)
		} else if m := regexp.MustCompile(`^VERSION_CODENAME=(.*)$`).FindStringSubmatch(s.Text()); m != nil {
			dist.versionCodename = strings.Trim(m[1], `"`)
		}
	}

	if dist.name == "" || dist.vendor == "" || dist.version == "" {
		return nil, fmt.Errorf("%+v: distribution %w", dist, errors.ErrUnknown)
	}

	dist.config = coreDistributionConfig

	vendor := strings.ToLower(dist.vendor)
	for _, sd := range sds {
		log.Tracef("Supported distribution: %v", sd.Name)

		if sd.Config == nil {
			continue
		}

		// nil vendors -> distribution configuration apply to all vendors
		if sd.Vendors == nil {
			dist.config.Merge(sd.Config)
			continue
		}

		// iterate distribution config vendors
		for _, distVendor := range sd.Vendors {
			if distVendor == vendor {
				log.Tracef("Use \"%v\" config (%v vendor match this config) ...", sd.Name, vendor)
				dist.config.Merge(sd.Config)
				break
			}
		}
	}

	if _, err := dist.config.Validate(); err != nil {
		return nil, err
	}

	return dist, nil
}

func (dist *distribution) Name() string {
	return dist.name
}

func (dist *distribution) Vendor() string {
	return dist.vendor
}

func (dist *distribution) Version() string {
	return dist.version
}

func (dist *distribution) VersionCodename() string {
	return dist.versionCodename
}

func (dist *distribution) InstallPackages(ctx context.Context, packages []string) error {

	log.Tracef("distribution.InstallPackages called with packages: %v", packages)

	if len(packages) == 0 {
		return nil
	}

	subCtx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	if err := dist.PackagesIndexUpdate(subCtx); err != nil {
		return err
	}

	if err := dist.PackagesInstallClean(subCtx, packages); err != nil {
		return err
	}

	return nil
}

func (dist *distribution) PackagesIndexUpdate(ctx context.Context) error {

	log.Trace("distribution.PackagesIndexUpdate called")

	return dist.execPackageManagerScript(ctx, dist.config.BinPackagesIndexUpdate)
}
func (dist *distribution) PackagesIndexClean(ctx context.Context) error {

	log.Trace("distribution.PackagesIndexClean called")

	return dist.execPackageManagerScript(ctx, dist.config.BinPackagesIndexClean)
}

func (dist *distribution) PackagesInstallClean(ctx context.Context, packages []string) error {

	log.Tracef("distribution.PackagesInstallClean called with packages: %+v", packages)

	return dist.execPackageManagerScript(ctx, dist.config.BinPackagesInstallClean, packages...)
}

func (dist *distribution) Config() *DistributionConfig {
	return dist.config
}

func (dist *distribution) execPackageManagerScript(ctx context.Context, script string, args ...string) error {

	log.Tracef("distribution.execPackageManagerScript called with script: %v %+v", script, args)

	return helpers.NewExec(ctx).Command(script, args...).Run()
}

type DistributionInstaller interface {
	Install(ctx context.Context) error
}

type distributionInstaller struct {
	dist Distribution
	fs   Filesystem
}

func newDistributionInstaller(fs Filesystem, dist Distribution) DistributionInstaller {
	return &distributionInstaller{
		dist: dist,
		fs:   fs,
	}
}

func (di *distributionInstaller) Install(ctx context.Context) error {

	log.Trace("distributionInstaller.Install called")

	log.Infof("Copying %v assets to container filesystem ...", di.dist.Name())
	for _, assets := range di.dist.Config().Assets {
		if err := di.copyAssets(assets); err != nil {
			return err
		}
	}

	log.Infof("Linking %v files to %v ...", di.fs.Paths().Bin, di.dist.Config().BinDest)
	if err := helpers.SymlinkAll(di.fs.Paths().Bin, di.dist.Config().BinDest); err != nil {
		return err
	}

	// exec container install.sh script
	installSh := filepath.Join(di.fs.Paths().Root, di.dist.Config().InstallScript)

	subCtx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	if err := helpers.NewExec(subCtx).Command(installSh).Run(); err != nil {
		return err
	}

	// remove container install.sh script
	if err := helpers.Remove(installSh); err != nil {
		return err
	}

	return nil
}

func (di *distributionInstaller) copyAssets(efs *embed.FS) error {

	log.Tracef("distributionInstaller.copyAssets called with efs: %+v", efs)

	if err := helpers.CopyEmbedDir(efs, di.fs.Paths().Root, di.assetPerm); err != nil {
		return err
	}

	return nil
}

func (di *distributionInstaller) assetPerm(file string) iofs.FileMode {

	log.Tracef("distributionInstaller.assetPerm called with file: %v", file)

	var perm iofs.FileMode = 0644

	// add execute to files in container bin directory and *.sh, *.sh.* files
	if strings.HasPrefix(file, di.fs.Paths().Bin) || strings.HasSuffix(file, ".sh") || strings.Contains(file, ".sh.") {
		perm = 0755
	}

	return perm
}
