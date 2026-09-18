// Command sdui-compile compiles every screen under an SDUI root into
// composed JSON files.
//
// Output layout:
//
//	<outDir>/
//	  manifest.json                    # per screen: versions map + etags
//	  screens/<id>/<templateVersion>.json
//
// A server (any language) can serve this output statically — the manifest
// carries everything needed for app-version selection and etag revalidation.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	sduicompiler "github.com/David-Lee-dev/sdui-template-compiler/go-cplr"
)

type cliOptions struct {
	rootDir string
	outDir  string
	pretty  bool
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	options, ok := parseArgs(argv)
	if !ok {
		fmt.Fprint(os.Stderr, "Usage: sdui-compile <rootDir> --out <outDir> [--pretty]\n")
		return 1
	}

	registry, err := sduicompiler.BuildScreenRegistry(options.rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	manifest := sduicompiler.NewObject()
	if err := os.MkdirAll(options.outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	for _, id := range registry.IDs() {
		module := registry.ModuleOf(id)
		if module == nil {
			continue
		}

		screenDir := filepath.Join(options.outDir, "screens", id)
		if err := os.MkdirAll(screenDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return 1
		}

		etags := sduicompiler.NewObject()
		written := map[string]struct{}{}
		for _, entry := range module.Versions {
			templateVersion := entry.Assets.Template
			if _, done := written[templateVersion]; done {
				continue
			}
			written[templateVersion] = struct{}{}

			composed, err := compiledTemplate(registry, id, templateVersion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return 1
			}
			serialized := sduicompiler.CompactJSON(composed.Template)
			if options.pretty {
				serialized = sduicompiler.PrettyJSON(composed.Template)
			}
			outPath := filepath.Join(screenDir, templateVersion+".json")
			if err := os.WriteFile(outPath, []byte(serialized), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return 1
			}
			etags.Set(templateVersion, composed.Etag)
		}

		versions := sduicompiler.NewObject()
		for _, entry := range module.Versions {
			assets := sduicompiler.NewObject()
			assets.Set("template", entry.Assets.Template)
			versions.Set(entry.Version, assets)
		}
		params := make([]sduicompiler.Value, 0, len(module.Params))
		for _, param := range module.Params {
			params = append(params, param)
		}

		screenManifest := sduicompiler.NewObject()
		screenManifest.Set("versions", versions)
		screenManifest.Set("params", params)
		screenManifest.Set("etags", etags)
		manifest.Set(id, screenManifest)
	}

	manifestPath := filepath.Join(options.outDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(sduicompiler.PrettyJSON(manifest)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "Compiled %d screen(s) to %s\n", len(registry.IDs()), options.outDir)
	return 0
}

func compiledTemplate(registry *sduicompiler.ScreenRegistry, id, templateVersion string) (*sduicompiler.ComposedTemplate, error) {
	module := registry.ModuleOf(id)
	if module == nil {
		return nil, fmt.Errorf("Unknown screen: %s", id)
	}

	// Resolve via the app-version threshold that maps to this template version.
	threshold := ""
	found := false
	for _, entry := range module.Versions {
		if entry.Assets.Template == templateVersion {
			threshold = entry.Version
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Screen <%s> has no threshold for %s", id, templateVersion)
	}

	composed, err := registry.Resolve(id, threshold)
	if err != nil {
		return nil, err
	}
	if composed == nil {
		return nil, fmt.Errorf("Screen <%s> failed to resolve %s", id, templateVersion)
	}
	return composed, nil
}

func parseArgs(argv []string) (cliOptions, bool) {
	var positional []string
	outDir := ""
	pretty := false

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "--out" || arg == "-o":
			if i+1 < len(argv) {
				outDir = argv[i+1]
			}
			i++
		case arg == "--pretty":
			pretty = true
		default:
			positional = append(positional, arg)
		}
	}

	if len(positional) != 1 || outDir == "" {
		return cliOptions{}, false
	}
	return cliOptions{rootDir: positional[0], outDir: outDir, pretty: pretty}, true
}
