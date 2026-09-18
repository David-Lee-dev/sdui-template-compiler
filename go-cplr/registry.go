package sduicompiler

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

// ComposedTemplate is one composed template and its content validator.
//
// The etag hashes the composed output — not the declared version string — so
// in-place edits inside one version directory still invalidate client caches.
type ComposedTemplate struct {
	Template Value
	Etag     string
}

type registeredScreen struct {
	module    *ScreenModule
	templates map[string]*ComposedTemplate
}

// etagLength truncates the content hash; two revisions of one screen
// colliding does not happen in practice.
const etagLength = 16

// ScreenRegistry serves composed templates by screen id and app version.
type ScreenRegistry struct {
	order   []string
	screens map[string]*registeredScreen
}

// BuildScreenRegistry composes and caches every template asset declared under
// an SDUI root. Screens are discovered from `screens/<dir>/screen.yaml`
// manifests; components resolve from `_components/` and tokens from
// `_tokens/`.
func BuildScreenRegistry(rootDir string) (registry *ScreenRegistry, err error) {
	defer recoverCompile(&err)
	return buildRegistry(rootDir, nil), nil
}

// BuildScreenRegistryWithModules is the explicit-modules override used by
// controlled test fixtures.
func BuildScreenRegistryWithModules(rootDir string, modules []*ScreenModule) (registry *ScreenRegistry, err error) {
	defer recoverCompile(&err)
	return buildRegistry(rootDir, modules), nil
}

func buildRegistry(rootDir string, modules []*ScreenModule) *ScreenRegistry {
	if modules == nil {
		modules = ScreenManifestDiscover(rootDir)
	}
	includeResolver := NewIncludeResolver(rootDir)
	tokenResolver := NewTokenResolver(rootDir)
	registry := &ScreenRegistry{screens: map[string]*registeredScreen{}}

	for _, module := range modules {
		if _, dup := registry.screens[module.ID]; dup {
			fail("Duplicate screen id: %s", module.ID)
		}

		templates := map[string]*ComposedTemplate{}
		for _, entry := range module.Versions {
			templateVersion := entry.Assets.Template
			if _, done := templates[templateVersion]; done {
				continue
			}
			rootPath := filepath.Join(module.Dir, "template", templateVersion, "_root.yaml")
			template := ComposerExpand(
				YamlLoad(rootPath),
				includeResolver,
				tokenResolver,
				rootDir,
				filepath.Dir(rootPath),
			)
			templates[templateVersion] = &ComposedTemplate{Template: template, Etag: etagOf(template)}
		}

		registry.order = append(registry.order, module.ID)
		registry.screens[module.ID] = &registeredScreen{module: module, templates: templates}
	}

	return registry
}

// Get returns the composed screen selected for an application version, or nil
// when the id is unknown. An empty appVersion selects the oldest threshold.
func (r *ScreenRegistry) Get(id, appVersion string) (Value, error) {
	composed, err := r.Resolve(id, appVersion)
	if composed == nil || err != nil {
		return nil, err
	}
	return composed.Template, nil
}

// Resolve returns the composed screen for an application version with its
// etag, or nil when the id is unknown. Errors when the application version is
// malformed.
func (r *ScreenRegistry) Resolve(id, appVersion string) (composed *ComposedTemplate, err error) {
	defer recoverCompile(&err)
	screen, ok := r.screens[id]
	if !ok {
		return nil, nil
	}
	assets := VersioningResolve(screen.module, appVersion)
	return screen.templates[assets.Template], nil
}

// IDs lists every registered screen id.
func (r *ScreenRegistry) IDs() []string {
	return append([]string{}, r.order...)
}

// VersionsOf lists app-version thresholds declared for a screen id, in
// ascending semantic order.
func (r *ScreenRegistry) VersionsOf(id string) []string {
	screen, ok := r.screens[id]
	if !ok {
		return []string{}
	}
	versions := make([]string, 0, len(screen.module.Versions))
	for _, entry := range screen.module.Versions {
		versions = append(versions, entry.Version)
	}
	for i := 1; i < len(versions); i++ {
		for j := i; j > 0 && CompareSemver(versions[j-1], versions[j]) > 0; j-- {
			versions[j-1], versions[j] = versions[j], versions[j-1]
		}
	}
	return versions
}

// ModuleOf returns the discovered module declaration for a screen id.
func (r *ScreenRegistry) ModuleOf(id string) *ScreenModule {
	screen, ok := r.screens[id]
	if !ok {
		return nil
	}
	return screen.module
}

func etagOf(template Value) string {
	sum := sha256.Sum256([]byte(CompactJSON(template)))
	return hex.EncodeToString(sum[:])[:etagLength]
}
