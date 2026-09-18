package sduicompiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const specDir = "../spec"

var fixturesDir = filepath.Join(specDir, "fixtures")

type versioningQuery struct {
	AppVersion *string `yaml:"app_version"`
	Template   string  `yaml:"template"`
}

type conformanceCases struct {
	Golden     []string `yaml:"golden"`
	Versioning struct {
		Fixture            string            `yaml:"fixture"`
		Screen             string            `yaml:"screen"`
		Queries            []versioningQuery `yaml:"queries"`
		InvalidAppVersions []string          `yaml:"invalid_app_versions"`
	} `yaml:"versioning"`
	Errors map[string]string `yaml:"errors"`
}

func loadCases(t *testing.T) conformanceCases {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(specDir, "cases.yaml"))
	if err != nil {
		t.Fatalf("read cases.yaml: %v", err)
	}
	var cases conformanceCases
	if err := yaml.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parse cases.yaml: %v", err)
	}
	return cases
}

func resolveTemplateVersion(t *testing.T, registry *ScreenRegistry, id, templateVersion string) *ComposedTemplate {
	t.Helper()
	module := registry.ModuleOf(id)
	if module == nil {
		t.Fatalf("Unknown screen: %s", id)
	}
	threshold := ""
	for _, entry := range module.Versions {
		if entry.Assets.Template == templateVersion {
			threshold = entry.Version
			break
		}
	}
	if threshold == "" {
		t.Fatalf("No threshold maps to template %s", templateVersion)
	}
	composed, err := registry.Resolve(id, threshold)
	if err != nil {
		t.Fatalf("resolve %s@%s: %v", id, threshold, err)
	}
	if composed == nil {
		t.Fatalf("Failed to resolve %s", id)
	}
	return composed
}

func TestConformanceGolden(t *testing.T) {
	cases := loadCases(t)
	for _, fixture := range cases.Golden {
		t.Run(fixture, func(t *testing.T) {
			input := filepath.Join(fixturesDir, fixture, "input")
			expectedDir := filepath.Join(fixturesDir, fixture, "expected")
			registry, err := BuildScreenRegistry(input)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			manifestBytes, err := os.ReadFile(filepath.Join(expectedDir, "manifest.json"))
			if err != nil {
				t.Fatalf("read expected manifest: %v", err)
			}
			var manifest map[string]struct {
				Etags map[string]string `json:"etags"`
			}
			if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
				t.Fatalf("parse expected manifest: %v", err)
			}

			for id, entry := range manifest {
				screenDir := filepath.Join(expectedDir, "screens", id)
				files, err := os.ReadDir(screenDir)
				if err != nil {
					t.Fatalf("read %s: %v", screenDir, err)
				}
				for _, file := range files {
					templateVersion := strings.TrimSuffix(file.Name(), ".json")
					composed := resolveTemplateVersion(t, registry, id, templateVersion)
					expected, err := os.ReadFile(filepath.Join(screenDir, file.Name()))
					if err != nil {
						t.Fatalf("read expected screen: %v", err)
					}
					if got := CompactJSON(composed.Template); got != string(expected) {
						t.Errorf("screen %s@%s JSON mismatch:\n got: %s\nwant: %s", id, templateVersion, got, expected)
					}
					if composed.Etag != entry.Etags[templateVersion] {
						t.Errorf("screen %s@%s etag = %s, want %s", id, templateVersion, composed.Etag, entry.Etags[templateVersion])
					}
				}
			}
		})
	}
}

func TestConformanceVersioning(t *testing.T) {
	cases := loadCases(t)
	spec := cases.Versioning
	input := filepath.Join(fixturesDir, spec.Fixture, "input")

	for _, query := range spec.Queries {
		label := "(none)"
		appVersion := ""
		if query.AppVersion != nil {
			label = *query.AppVersion
			appVersion = *query.AppVersion
		}
		t.Run("selects "+query.Template+" for "+label, func(t *testing.T) {
			registry, err := BuildScreenRegistry(input)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			composed, err := registry.Resolve(spec.Screen, appVersion)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			viaThreshold := resolveTemplateVersion(t, registry, spec.Screen, query.Template)
			if composed == nil || composed.Etag != viaThreshold.Etag {
				t.Errorf("resolve(%s, %q) selected wrong template", spec.Screen, appVersion)
			}
		})
	}

	for _, invalid := range spec.InvalidAppVersions {
		t.Run("rejects "+invalid, func(t *testing.T) {
			registry, err := BuildScreenRegistry(input)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if _, err := registry.Resolve(spec.Screen, invalid); err == nil || !strings.Contains(err.Error(), "Invalid semantic version") {
				t.Errorf("resolve(%s, %q) error = %v, want Invalid semantic version", spec.Screen, invalid, err)
			}
		})
	}
}

func TestConformanceErrors(t *testing.T) {
	cases := loadCases(t)
	for fixture, message := range cases.Errors {
		t.Run(fixture, func(t *testing.T) {
			input := filepath.Join(fixturesDir, "errors", fixture, "input")
			_, err := BuildScreenRegistry(input)
			if err == nil || !strings.Contains(err.Error(), message) {
				t.Errorf("build error = %v, want message containing %q", err, message)
			}
		})
	}
}
