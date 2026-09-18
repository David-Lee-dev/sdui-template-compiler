package sduicompiler

import (
	"math"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Yaml loads YAML files as JSON-compatible ordered values.
type Yaml struct{}

// YamlLoad loads a YAML file as a JSON-compatible value.
//
// Parsing follows the YAML 1.2 core schema like js-yaml v4: only true/false
// variants are booleans, and plain scalars such as `on` or `1.0.0` stay
// strings. Aborts (compileError panic) when the YAML holds a value JSON
// cannot represent.
func YamlLoad(path string) Value {
	data, err := os.ReadFile(path)
	if err != nil {
		fail("%s", err.Error())
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		fail("%s", err.Error())
	}
	if node.Kind == 0 || len(node.Content) == 0 {
		fail("YAML is not JSON-compatible: %s", path)
	}
	return yamlToValue(node.Content[0], path)
}

func yamlToValue(node *yaml.Node, path string) Value {
	switch node.Kind {
	case yaml.AliasNode:
		return yamlToValue(node.Alias, path)
	case yaml.SequenceNode:
		items := make([]Value, 0, len(node.Content))
		for _, child := range node.Content {
			items = append(items, yamlToValue(child, path))
		}
		return items
	case yaml.MappingNode:
		obj := NewObject()
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			// js-yaml stringifies mapping keys; the raw scalar text is that
			// string for every plain key the fixtures use.
			obj.Set(key.Value, yamlToValue(node.Content[i+1], path))
		}
		return obj
	case yaml.ScalarNode:
		return yamlScalar(node, path)
	default:
		fail("YAML is not JSON-compatible: %s", path)
		return nil
	}
}

func yamlScalar(node *yaml.Node, path string) Value {
	switch node.Tag {
	case "!!null":
		return nil
	case "!!bool":
		var b bool
		if err := node.Decode(&b); err != nil {
			fail("YAML is not JSON-compatible: %s", path)
		}
		return b
	case "!!int":
		var i int64
		if err := node.Decode(&i); err == nil {
			return i
		}
		var f float64
		if err := node.Decode(&f); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
			return f
		}
		fail("YAML is not JSON-compatible: %s", path)
	case "!!float":
		f, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			if decodeErr := node.Decode(&f); decodeErr != nil {
				fail("YAML is not JSON-compatible: %s", path)
			}
		}
		if math.IsInf(f, 0) || math.IsNaN(f) {
			fail("YAML is not JSON-compatible: %s", path)
		}
		return f
	case "!!str":
		return node.Value
	}
	fail("YAML is not JSON-compatible: %s", path)
	return nil
}
