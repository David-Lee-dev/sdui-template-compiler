package sduicompiler

import (
	"sort"
	"strconv"
)

type semanticVersion struct {
	major, minor, patch int
	source              string
}

// VersioningResolve resolves the assets declared by the newest supported
// app-version threshold. An empty appVersion selects the oldest threshold.
// Aborts when the application version is malformed.
func VersioningResolve(module *ScreenModule, appVersion string) ScreenVersionAssets {
	versions := make([]semanticVersion, 0, len(module.Versions))
	for _, entry := range module.Versions {
		versions = append(versions, parseSemver(entry.Version))
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return compareSemverParsed(versions[i], versions[j]) < 0
	})

	if appVersion == "" {
		assets, _ := module.VersionAssets(versions[0].source)
		return assets
	}

	client := parseSemver(appVersion)
	threshold := versions[0].source
	for _, version := range versions {
		if compareSemverParsed(version, client) <= 0 {
			threshold = version.source
		}
	}
	assets, _ := module.VersionAssets(threshold)
	return assets
}

// CompareSemver compares two strict major.minor.patch semantic versions.
func CompareSemver(left, right string) int {
	return compareSemverParsed(parseSemver(left), parseSemver(right))
}

func parseSemver(version string) semanticVersion {
	match := semanticVersionPattern.FindStringSubmatch(version)
	if match == nil {
		fail("Invalid semantic version: %s", version)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	return semanticVersion{major: major, minor: minor, patch: patch, source: version}
}

func compareSemverParsed(left, right semanticVersion) int {
	if d := left.major - right.major; d != 0 {
		return d
	}
	if d := left.minor - right.minor; d != 0 {
		return d
	}
	return left.patch - right.patch
}
