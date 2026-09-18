package sduicompiler

import (
	"sort"
	"strings"
)

type semanticVersion struct {
	major, minor, patch string
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
	return semanticVersion{major: match[1], minor: match[2], patch: match[3], source: version}
}

// compareComponent compares two version components as digit strings. The
// regex forbids leading zeros, so length-then-lexicographic equals numeric
// order at arbitrary precision — machine ints would overflow silently.
func compareComponent(left, right string) int {
	if len(left) != len(right) {
		return len(left) - len(right)
	}
	return strings.Compare(left, right)
}

func compareSemverParsed(left, right semanticVersion) int {
	if d := compareComponent(left.major, right.major); d != 0 {
		return d
	}
	if d := compareComponent(left.minor, right.minor); d != 0 {
		return d
	}
	return compareComponent(left.patch, right.patch)
}
