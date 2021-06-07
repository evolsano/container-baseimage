package cmd

import (
	"github.com/osixia/container-baseimage/build/job"
)

type deployFlags struct {
	testFlags

	dryRun bool
}

var deployCmdFlags = &deployFlags{}

var deployCmd = newStepCmd("deploy", "Run build, test and deploy jobs", "d", job.Deploy, deployCmdFlags)

func init() {
	// flags
	deployCmd.Flags().SortFlags = false

	deployCmd.Flags().BoolVarP(&deployCmdFlags.dryRun, "dry-run", "d", false, "do not deploy images to registry\n")
	addBuildFlags(deployCmd.Flags(), &deployCmdFlags.buildFlags)
}

func (df *deployFlags) toJobOptions() interface{} {

	return &job.DeployOptions{
		TestOptions: *df.testFlags.toJobOptions().(*job.TestOptions),

		DryRun: df.dryRun,
	}
}
