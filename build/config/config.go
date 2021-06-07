package config

import "dagger.io/dagger"

type Image struct {
	RootImage    string
	Distribution string

	BuildImageName string
	TagPrefixes    []string
}

type Platform struct {
	Name   dagger.Platform
	GoArch string
}

type GithubRepo struct {
	Organization string
	Project      string
}

type Nonroot struct {
	UserName  string
	UserID    int
	GroupName string
	GroupID   int
}

// images

var DefaultVersion = "develop"
var DefaultImage = DebianBookwormImage

var DebianBookwormImage = &Image{
	RootImage:    "debian:bookworm-slim",
	Distribution: "debian",

	BuildImageName: "osixia/baseimage",
	TagPrefixes:    []string{"debian-bookworm", "debian"},
}

var Ubuntu2404Image = &Image{
	RootImage:    "ubuntu:24.04",
	Distribution: "ubuntu",

	BuildImageName: "osixia/baseimage",
	TagPrefixes:    []string{"ubuntu-24.04", "ubuntu"},
}

var Alpine321Image = &Image{
	RootImage:    "alpine:3.21.2",
	Distribution: "alpine",

	BuildImageName: "osixia/baseimage",
	TagPrefixes:    []string{"alpine-3.21", "alpine-3", "alpine"},
}

var Images = []*Image{
	DebianBookwormImage,
	Ubuntu2404Image,
	Alpine321Image,
}

// platforms

var Amd64Platform = &Platform{
	Name:   "linux/amd64",
	GoArch: "amd64",
}

var Arm64Platform = &Platform{
	Name:   "linux/arm64",
	GoArch: "arm64",
}

var Platforms = []*Platform{
	Amd64Platform,
	Arm64Platform,
}

// nonroot

var NonrootUser = "nonroot"
var NonrootUID = 65532

var NonrootGroup = "nonroot"
var NonrootGID = 65532

// github repo

var BaseimageGithubRepo = &GithubRepo{
	Organization: "osixia",
	Project:      "container-baseimage",
}
