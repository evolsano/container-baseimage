package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"go.uber.org/thriftrw/ptr"

	"github.com/osixia/container-baseimage/build/config"
	"github.com/osixia/container-baseimage/build/job"
)

type githubFlags struct {
	dryRun bool
}

type githubData struct {
	contributors []string
	tags         []string
}

var githubCmdFlags = &githubFlags{}

var githubCmd = &cobra.Command{
	Use:   "github dockerfile github_ref",
	Short: "Run jobs based on GitHub Actions CI/CD environment",

	GroupID: pipelineGroupID,

	Args: cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {

		dockerfile := args[0]
		githubRef := args[1]

		fmt.Printf("Dockerfile: %v, githubRef: %v\n", dockerfile, githubRef)

		testFlags := testFlags{
			buildFlags: buildFlags{
				dockerfile: dockerfile,

				distributions: distributions(),
				arches:        arches(),

				withNonroot: true,
			},
		}

		jb := job.Test
		var jFlags jobFlags = &testFlags

		// GITHUB_REF values:
		// refs/heads/<branch_name>, refs/pull/<pr_number>/merge, refs/tags/<tag_name>

		// Build and test: branches main, develop, feature/*, bugfix/*, release/*, hotfix/*,  support/*
		// Build, test and deploy: tags

		ref := regexp.MustCompile(`^refs/(heads|tags|pull)/(.*)$`).FindStringSubmatch(githubRef)
		if ref == nil || (len(ref) != 3) {
			fatal(fmt.Errorf("unable to get github reference type and name %v", ref))
		}

		refType := ref[1]
		refName := ref[2]

		testFlags.version = refName

		if refType == "pull" {

			testFlags.version = strings.TrimSuffix(refName, "/merge")

		} else if refType == "tags" { // deploy

			if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+-?[^.]*$`).MatchString(refName) {
				fatal(fmt.Errorf("%v tag must be formated like x.y.z or x.y.z-a with x, y and z numbers and a any char except '.'", refName))
			}

			gi, err := githubDataRequest(config.BaseimageGithubRepo)
			if err != nil {
				fatal(err)
			}

			testFlags.latest, err = isLatestTag(testFlags.version, gi.tags)
			if err != nil {
				fatal(err)
			}

			deployFlags := deployFlags{
				testFlags: testFlags,

				dryRun: githubCmdFlags.dryRun,
			}
			deployFlags.contributors = strings.Join(gi.contributors, ", ")

			jb = job.Deploy
			jFlags = &deployFlags

		}

		brs, err := job.Run(cmd.Context(), jb, jFlags.toJobOptions())
		if err != nil {
			fatal(err)
		}

		if jb == job.Deploy {
			if err := githubReleaseRequest(cmd.Context(), config.BaseimageGithubRepo, jFlags.(*deployFlags), brs); err != nil {
				fatal(err)
			}
		}

	},
}

func init() {
	// flags
	githubCmd.Flags().SortFlags = false
	githubCmd.Flags().BoolVarP(&githubCmdFlags.dryRun, "dry-run", "d", false, "do not deploy images to registry\n")
}

func isLatestTag(tag string, existingTags []string) (bool, error) {

	// test latest only on x.y.z tags (not x.y.z-a)
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tag) {
		return false, nil
	}

	tVersion, err := version.NewVersion(tag)
	if err != nil {
		return false, err
	}

	// remove prefixes from github tags
	etVersions := make([]*version.Version, 0, len(existingTags))
	for _, gtag := range existingTags {
		m := regexp.MustCompile(`^.*([0-9]+\.[0-9]+\.[0-9]+)-?.*$`).FindStringSubmatch(gtag)
		if len(m) < 2 {
			return false, fmt.Errorf("error: parsing tag: %v", gtag)
		}

		v, err := version.NewVersion(m[1])
		if err != nil {
			return false, err
		}
		etVersions = append(etVersions, v)
	}

	// sort github tags
	sort.Sort(version.Collection(etVersions))

	// compare new tag version and greated github tag version
	return tVersion.GreaterThanOrEqual(etVersions[len(etVersions)-1]), nil
}

func githubDataRequest(repo *config.GithubRepo) (*githubData, error) {

	client := github.NewClient(nil)

	// get all contributors
	copt := &github.ListContributorsOptions{}

	var contributors []string
	for {
		cbs, r, err := client.Repositories.ListContributors(cmd.Context(), repo.Organization, repo.Project, copt)
		if err != nil {
			return nil, err
		}

		for _, c := range cbs {
			contributors = append(contributors, *c.Login)
		}

		if r.NextPage == 0 {
			break
		}

		copt.Page = r.NextPage
	}

	// get all tags
	topt := &github.ListOptions{}

	var tags []string
	for {

		ts, r, err := client.Repositories.ListTags(context.Background(), repo.Organization, repo.Project, topt)
		if err != nil {
			return nil, err
		}

		for _, t := range ts {
			tags = append(tags, *t.Name)
		}

		if r.NextPage == 0 {
			break
		}

		topt.Page = r.NextPage
	}

	// sort results
	sort.Strings(tags)

	return &githubData{
		contributors: contributors,
		tags:         tags,
	}, nil
}

func githubReleaseRequest(ctx context.Context, repo *config.GithubRepo, df *deployFlags, brs []*job.BuildResult) error {

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return errors.New("environment variable GITHUB_TOKEN must be set")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	fmt.Printf("Working on release %v", df.version)
	release := &github.RepositoryRelease{
		TagName: ptr.String(df.version),
		Name:    ptr.String(df.version),
		Body:    ptr.String("See CHANGELOG.md"),
	}

	var previousRelease *github.RepositoryRelease

	// check if the release already exists
	releases, _, err := client.Repositories.ListReleases(ctx, repo.Organization, repo.Project, nil)
	if err != nil {
		return err
	}
	for _, r := range releases {
		if r.GetTagName() == release.GetTagName() {
			previousRelease = r
		}
	}

	if previousRelease != nil {
		fmt.Printf("Updating existing %v release", previousRelease.GetTagName())
		if !df.dryRun {
			release, _, err = client.Repositories.EditRelease(ctx, repo.Organization, repo.Project, previousRelease.GetID(), release)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Printf("Create release %v", release.GetTagName())
		if !df.dryRun {
			release, _, err = client.Repositories.CreateRelease(ctx, repo.Organization, repo.Project, release)
			if err != nil {
				return err
			}
		}
	}

	for _, br := range brs {
		for _, f := range br.Files {
			file, err := os.Open(f)
			if err != nil {
				return fmt.Errorf("error opening %s: %w", f, err)
			}
			defer file.Close()

			if df.dryRun {
				fmt.Printf("Would have uploaded %v", file.Name())
				continue
			}

			opts := &github.UploadOptions{Name: file.Name()}
			asset, _, err := client.Repositories.UploadReleaseAsset(ctx, repo.Organization, repo.Project, release.GetID(), opts, file)
			if err != nil {
				return fmt.Errorf("error uploading release asset: %w", err)
			}

			fmt.Printf("Asset uploaded: %v", asset.GetBrowserDownloadURL())
		}
	}

	return nil
}
