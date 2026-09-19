package sduicompiler

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	refKey      = ".ref"
	varsKey     = ".vars"
	defaultsKey = ".defaults"
	contentKey  = ".content"
	tokenKey    = ".token"
)

// missingSlot marks a component variable slot omitted by the caller.
type missingSentinel struct{}

var missingSlot = missingSentinel{}

var tokenGroupPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// tokenSigilPattern matches `@{group.path}` — the inline form of `.token` —
// and `@@{`, its escape. Shared shape with the other ports.
var tokenSigilPattern = regexp.MustCompile(`@@\{|@\{([^{}]*)\}`)

// componentFile is a parsed component declaration (.vars/.defaults/.content).
type componentFile struct {
	vars     []string
	defaults *Object
	content  Value
}

type expansionContext struct {
	activePaths     []string
	componentStack  []string
	includeResolver IncludeResolverFn
	includeStack    []string
	referrerDir     string
	rootDir         string
	tokenResolver   TokenResolverFn
}

// ComposerExpand expands a parsed screen body into client-owned JSON.
//
// Mirrors the TS Composer.expand: `.ref` components/fragments, `.token`
// resolution, slot substitution, array slot splicing, and circular
// detection — with component args expanded in the caller's context before
// the target file is pushed onto the active chain.
func ComposerExpand(
	rootBody Value,
	includeResolver IncludeResolverFn,
	tokenResolver TokenResolverFn,
	rootDir string,
	referrerDir string,
) Value {
	return expandNode(rootBody, expansionContext{
		includeResolver: includeResolver,
		referrerDir:     referrerDir,
		rootDir:         absPath(rootDir),
		tokenResolver:   tokenResolver,
	})
}

func expandNode(node Value, context expansionContext) Value {
	if arr, ok := node.([]Value); ok {
		return expandArray(arr, context)
	}
	if s, ok := node.(string); ok {
		return expandString(s, context)
	}
	obj, ok := node.(*Object)
	if !ok {
		return node
	}

	if obj.Has("screen_id") {
		fail("screen_id is not allowed in template nodes")
	}
	if obj.Has(tokenKey) {
		return expandToken(obj, context)
	}
	if obj.Has(refKey) {
		return expandReference(obj, context)
	}

	result := NewObject()
	for _, key := range obj.Keys() {
		if strings.HasPrefix(key, ".") {
			fail("Unsupported build key %s", key)
		}
		value, _ := obj.Get(key)
		result.Set(key, expandNode(value, context))
	}
	return result
}

func expandToken(node *Object, context expansionContext) Value {
	if node.Len() != 1 {
		fail(".token node must not have sibling keys")
	}

	rawPath, _ := node.Get(tokenKey)
	tokenPath, ok := rawPath.(string)
	if !ok {
		fail(".token value must be a non-empty dotted string")
	}

	return resolveToken(tokenPath, context)
}

// resolveToken resolves a dotted token path against its group file.
//
// Shared by `{ .token: ... }` and the inline `@{...}` sigil so both forms
// validate identically and fail with the same diagnostics.
func resolveToken(tokenPath string, context expansionContext) Value {
	segments := strings.Split(tokenPath, ".")
	valid := len(segments) >= 2 && tokenGroupPattern.MatchString(segments[0])
	for _, segment := range segments {
		if len(segment) == 0 || strings.TrimSpace(segment) != segment {
			valid = false
		}
	}
	if !valid {
		fail(".token value must be a non-empty dotted string")
	}

	group, keys := segments[0], segments[1:]
	var value Value = context.tokenResolver(group)
	traversedPath := group

	for _, key := range keys {
		obj, ok := value.(*Object)
		if !ok {
			fail("Cannot traverse non-object token path %s while resolving %s", traversedPath, tokenPath)
		}
		next, has := obj.Get(key)
		if !has {
			fail("Missing token key %s at %s.%s", tokenPath, traversedPath, key)
		}
		value = next
		traversedPath = traversedPath + "." + key
	}

	// A token value that carries the sigil would be rescanned when it lands in
	// a component argument subtree, making resolution depend on where it was
	// used. Reject it at the source instead.
	if s, ok := value.(string); ok && strings.Contains(s, "@{") {
		fail("Token value must not contain @{: %s", tokenPath)
	}

	return cloneValue(value)
}

// expandString substitutes `@{group.path}` token sigils inside a string value.
//
// A string that is exactly one sigil resolves to the token's own value and
// keeps its type (`'@{spacing.lg}'` -> `16`). Any other occurrence is
// interpolated as text, so tokens compose inside runtime expressions.
func expandString(node string, context expansionContext) Value {
	if !strings.Contains(node, "@") {
		return node
	}

	if match := tokenSigilPattern.FindStringSubmatchIndex(node); match != nil &&
		match[0] == 0 && match[1] == len(node) && match[2] >= 0 {
		return resolveToken(node[match[2]:match[3]], context)
	}

	assertNoDanglingSigil(node)

	var sb strings.Builder
	last := 0
	for _, match := range tokenSigilPattern.FindAllStringSubmatchIndex(node, -1) {
		sb.WriteString(node[last:match[0]])
		last = match[1]

		if match[2] < 0 {
			sb.WriteString("@{")
			continue
		}

		path := node[match[2]:match[3]]
		value := resolveToken(path, context)
		switch t := value.(type) {
		case string:
			sb.WriteString(t)
		case []Value, *Object:
			fail("Cannot interpolate non-scalar token %s into a string", path)
		default:
			sb.WriteString(CompactJSON(value))
		}
	}
	sb.WriteString(node[last:])
	return sb.String()
}

// assertNoDanglingSigil rejects an unterminated `@{` left behind by
// substitution. A typo like `@{color.primary` would otherwise ship to the
// client as literal text; failing the build is the loud alternative.
func assertNoDanglingSigil(original string) {
	if strings.Contains(tokenSigilPattern.ReplaceAllString(original, ""), "@{") {
		fail("Unterminated token sigil @{ in: %s", original)
	}
}

func expandArray(nodes []Value, context expansionContext) []Value {
	result := make([]Value, 0, len(nodes))
	for _, node := range nodes {
		expanded := expandNode(node, context)
		if isReference(node) {
			if spliced, ok := expanded.([]Value); ok {
				result = append(result, spliced...)
				continue
			}
		}
		result = append(result, expanded)
	}
	return result
}

func expandReference(node *Object, context expansionContext) Value {
	if node.Has("_type") {
		fail(".ref node must not declare _type")
	}

	rawRef, _ := node.Get(refKey)
	refPath, ok := rawRef.(string)
	if !ok {
		fail(".ref path must be a string")
	}

	resolvedInclude := context.includeResolver(refPath, context.referrerDir)
	filePath := resolveAgainst(context.rootDir, resolvedInclude.FilePath)
	validatorAssertNoCircularPath(filePath, context.activePaths)

	// Component args are authored by the caller, so expand them in the caller's
	// context — before this file is pushed onto activePaths. Otherwise reusing a
	// component inside another instance's arg subtree (e.g. a /card in a /card's
	// child) re-enters the same file while it is still "active" and is misflagged
	// as circular. Genuine self-recursion is still caught: a component whose own
	// .content re-references itself expands that .content with the file on the path.
	args := NewObject()
	for _, key := range node.Keys() {
		if key == refKey {
			continue
		}
		value, _ := node.Get(key)
		args.Set(key, expandNode(value, context))
	}

	targetContext := context
	targetContext.activePaths = appendPath(context.activePaths, filePath)
	targetContext.referrerDir = filepath.Dir(filePath)

	if target, ok := resolvedInclude.Content.(*Object); ok && target.Has(varsKey) {
		return expandComponent(target, args, filePath, targetContext)
	}

	if args.Len() > 0 {
		fail("Fragment %s does not accept args: %s", filePath, strings.Join(args.Keys(), ", "))
	}

	fragmentContext := targetContext
	fragmentContext.includeStack = appendPath(context.includeStack, filePath)
	return expandNode(resolvedInclude.Content, fragmentContext)
}

func expandComponent(target *Object, args *Object, filePath string, context expansionContext) Value {
	component := toComponentFile(target, filePath)
	validatorAssertArgsDeclared(component.vars, args)

	effectiveArgs := NewObject()
	if component.defaults != nil {
		for _, key := range component.defaults.Keys() {
			value, _ := component.defaults.Get(key)
			effectiveArgs.Set(key, value)
		}
	}
	for _, key := range args.Keys() {
		value, _ := args.Get(key)
		effectiveArgs.Set(key, value)
	}

	substituted := substituteNode(component.content, component.vars, effectiveArgs)
	if _, omitted := substituted.(missingSentinel); omitted {
		fail("Component body resolved to an omitted slot: %s", filePath)
	}

	componentContext := context
	componentContext.componentStack = appendPath(context.componentStack, filePath)
	return expandNode(substituted, componentContext)
}

// spliceSlots substitutes array slots while splicing list-valued arguments in
// place and dropping omitted slots.
func spliceSlots(nodes []Value, vars []string, args *Object) []Value {
	result := make([]Value, 0, len(nodes))

	for _, node := range nodes {
		name, isSlot := slotName(node)
		if isSlot {
			validatorAssertHoleDeclared(vars, name)
			value, has := args.Get(name)
			if !has {
				continue
			}
			cloned := cloneValue(value)
			if list, ok := cloned.([]Value); ok {
				result = append(result, list...)
			} else {
				result = append(result, cloned)
			}
			continue
		}

		substituted := substituteNode(node, vars, args)
		if _, omitted := substituted.(missingSentinel); !omitted {
			result = append(result, substituted)
		}
	}

	return result
}

func substituteNode(node Value, vars []string, args *Object) Value {
	if arr, ok := node.([]Value); ok {
		return spliceSlots(arr, vars, args)
	}

	if obj, ok := node.(*Object); ok {
		result := NewObject()
		for _, key := range obj.Keys() {
			if strings.HasPrefix(key, ".") && key != refKey && key != tokenKey {
				fail("Unsupported build key %s", key)
			}
			value, _ := obj.Get(key)
			substituted := substituteNode(value, vars, args)
			if _, omitted := substituted.(missingSentinel); !omitted {
				result.Set(key, substituted)
			}
		}
		return result
	}

	name, isSlot := slotName(node)
	if !isSlot {
		return node
	}

	validatorAssertHoleDeclared(vars, name)
	value, has := args.Get(name)
	if !has {
		return missingSlot
	}
	return cloneValue(value)
}

func toComponentFile(target *Object, filePath string) componentFile {
	if target.Has("screen_id") {
		fail("screen_id is not allowed in template nodes")
	}
	for _, key := range target.Keys() {
		if strings.HasPrefix(key, ".") && key != varsKey && key != defaultsKey && key != contentKey {
			fail("Unsupported build key %s", key)
		}
	}

	rawVars, _ := target.Get(varsKey)
	vars, ok := toStringSlice(rawVars)
	if !ok || containsEmpty(vars) {
		fail("Component .vars must be a list of names: %s", filePath)
	}
	if hasDuplicates(vars) {
		fail("Component .vars contains duplicate names: %s", filePath)
	}

	var defaults *Object
	if rawDefaults, has := target.Get(defaultsKey); has {
		obj, ok := rawDefaults.(*Object)
		if !ok {
			fail("Component .defaults must be a map: %s", filePath)
		}
		for _, key := range obj.Keys() {
			if !containsString(vars, key) {
				fail("Component .defaults key not in .vars: %s (%s)", key, filePath)
			}
		}
		defaults = obj
	}

	if !target.Has(contentKey) {
		fail("Component with .vars must declare .content: %s", filePath)
	}

	content, _ := target.Get(contentKey)
	return componentFile{vars: vars, defaults: defaults, content: content}
}

// slotName returns the variable name of a slot node (a `.name` string that is
// not a relative path), or ok=false when the node is not a slot.
func slotName(node Value) (string, bool) {
	s, ok := node.(string)
	if !ok || !strings.HasPrefix(s, ".") || len(s) == 1 {
		return "", false
	}
	if strings.HasPrefix(s, "./") || strings.HasPrefix(s, "..") {
		return "", false
	}
	return s[1:], true
}

func isReference(node Value) bool {
	obj, ok := node.(*Object)
	return ok && obj.Has(refKey)
}

func toStringSlice(node Value) ([]string, bool) {
	arr, ok := node.([]Value)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

func containsEmpty(items []string) bool {
	for _, item := range items {
		if item == "" {
			return true
		}
	}
	return false
}

func hasDuplicates(items []string) bool {
	seen := map[string]struct{}{}
	for _, item := range items {
		if _, ok := seen[item]; ok {
			return true
		}
		seen[item] = struct{}{}
	}
	return false
}

func cloneValue(node Value) Value {
	switch t := node.(type) {
	case []Value:
		out := make([]Value, len(t))
		for i, item := range t {
			out[i] = cloneValue(item)
		}
		return out
	case *Object:
		out := NewObject()
		for _, key := range t.Keys() {
			value, _ := t.Get(key)
			out.Set(key, cloneValue(value))
		}
		return out
	default:
		return node
	}
}

// resolveAgainst mirrors Node's path.resolve(base, p): an absolute p wins.
func resolveAgainst(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}
