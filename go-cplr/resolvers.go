package sduicompiler

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvedInclude is a resolved component or fragment reference.
type ResolvedInclude struct {
	FilePath string
	Content  Value
}

// IncludeResolverFn resolves a `.ref` path from a referrer directory.
type IncludeResolverFn func(refPath, referrerDir string) ResolvedInclude

// TokenResolverFn resolves a design-token group to its map.
type TokenResolverFn func(group string) *Object

// NewIncludeResolver creates a disk-backed resolver for component and
// fragment references. Absolute-style refs (`/x`) resolve from
// `<rootDir>/_components`; relative refs resolve from the referrer directory.
// Aborts when a referenced file is missing or is not JSON-compatible.
func NewIncludeResolver(rootDir string) IncludeResolverFn {
	root := absPath(rootDir)
	return func(refPath, referrerDir string) ResolvedInclude {
		var filePath string
		if strings.HasPrefix(refPath, "/") {
			filePath = filepath.Join(root, "_components", refPath[1:]+".yaml")
		} else {
			filePath = filepath.Join(absPath(referrerDir), refPath+".yaml")
		}
		if !fileExists(filePath) {
			fail("Unresolved .ref %s: %s", refPath, filePath)
		}
		return ResolvedInclude{FilePath: filePath, Content: YamlLoad(filePath)}
	}
}

// NewTokenResolver creates a cached disk-backed resolver for design-token
// groups under `<rootDir>/_tokens`. Aborts when a group file is missing or
// does not contain a map.
func NewTokenResolver(rootDir string) TokenResolverFn {
	root := absPath(rootDir)
	cache := map[string]*Object{}
	return func(group string) *Object {
		if cached, ok := cache[group]; ok {
			return cached
		}
		filePath := filepath.Join(root, "_tokens", group+".yaml")
		if !fileExists(filePath) {
			fail("Unknown token group %s: %s", group, filePath)
		}
		tokens, ok := YamlLoad(filePath).(*Object)
		if !ok {
			fail("Token group %s must contain a map: %s", group, filePath)
		}
		cache[group] = tokens
		return tokens
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		fail("%s", err.Error())
	}
	return abs
}
