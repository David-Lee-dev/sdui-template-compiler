package sduicompiler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ScreenVersionAssets is the asset set one app-version threshold declares.
type ScreenVersionAssets struct {
	Template string
}

// VersionEntry keeps the version map ordered as authored in screen.yaml.
type VersionEntry struct {
	Version string
	Assets  ScreenVersionAssets
}

// ScreenModule is a screen declaration loaded from `screens/<dir>/screen.yaml`.
type ScreenModule struct {
	ID string
	// Dir is the absolute path to the screen directory holding its templates.
	Dir string
	// Versions maps app-version thresholds to assets, in declaration order.
	Versions []VersionEntry
	// Params are route query-parameter keys the template may bind as root data.
	Params []string
}

// VersionAssets returns the assets for one threshold key.
func (m *ScreenModule) VersionAssets(version string) (ScreenVersionAssets, bool) {
	for _, entry := range m.Versions {
		if entry.Version == version {
			return entry.Assets, true
		}
	}
	return ScreenVersionAssets{}, false
}

var semanticVersionPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// JS objects reorder integer-index keys ("2" before "10" regardless of
// insertion), so purely numeric ids/versions would serialize in different
// manifest orders across ports. Rejected at load time instead.
var purelyNumericPattern = regexp.MustCompile(`^\d+$`)

const manifestFile = "screen.yaml"

// ScreenManifestDiscover discovers every screen manifest under
// `<rootDir>/screens`, in lexicographic directory-name order (os.ReadDir
// sorts). Aborts when a manifest is invalid
// or two screens declare the same id.
func ScreenManifestDiscover(rootDir string) []*ScreenModule {
	screensDir := filepath.Join(rootDir, "screens")
	entries, err := os.ReadDir(screensDir)
	if err != nil {
		fail("Missing screens directory: %s", screensDir)
	}

	var modules []*ScreenModule
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(screensDir, entry.Name(), manifestFile)
		if !fileExists(manifestPath) {
			continue
		}

		module := ScreenManifestLoad(manifestPath, filepath.Join(screensDir, entry.Name()))
		if _, dup := seen[module.ID]; dup {
			fail("Duplicate screen id: %s", module.ID)
		}
		seen[module.ID] = struct{}{}
		modules = append(modules, module)
	}
	return modules
}

// ScreenManifestLoad loads and validates a single screen manifest file.
// Aborts when the id, version map, or a version threshold is invalid.
func ScreenManifestLoad(manifestPath, dir string) *ScreenModule {
	raw, ok := YamlLoad(manifestPath).(*Object)
	if !ok {
		fail("Screen manifest must be a map: %s", manifestPath)
	}

	rawID, _ := raw.Get("id")
	id, ok := rawID.(string)
	if !ok || strings.TrimSpace(id) == "" {
		fail("Screen manifest must declare an id: %s", manifestPath)
	}
	if purelyNumericPattern.MatchString(id) {
		fail("Screen id must not be purely numeric: %s", id)
	}

	rawVersions, _ := raw.Get("versions")
	versionsMap, ok := rawVersions.(*Object)
	if !ok {
		fail("Screen <%s> must declare a versions map", id)
	}
	if versionsMap.Len() == 0 {
		fail("Screen <%s> must declare at least one version", id)
	}

	var versions []VersionEntry
	for _, version := range versionsMap.Keys() {
		if !semanticVersionPattern.MatchString(version) {
			fail("Invalid semantic version: %s", version)
		}
		rawAssets, _ := versionsMap.Get(version)
		assets, ok := rawAssets.(*Object)
		if !ok {
			fail("Screen <%s> version %s must be a map", id, version)
		}
		rawTemplate, _ := assets.Get("template")
		template, ok := rawTemplate.(string)
		if !ok || template == "" {
			fail("Screen <%s> version %s must declare a template", id, version)
		}
		if purelyNumericPattern.MatchString(template) {
			fail("Template version must not be purely numeric: %s", template)
		}
		versions = append(versions, VersionEntry{Version: version, Assets: ScreenVersionAssets{Template: template}})
	}

	params := []string{}
	if rawParams, has := raw.Get("params"); has && rawParams != nil {
		list, ok := toStringSlice(rawParams)
		if !ok {
			fail("Screen <%s> params must be a list of names", id)
		}
		params = list
	}

	return &ScreenModule{ID: id, Dir: dir, Versions: versions, Params: params}
}
