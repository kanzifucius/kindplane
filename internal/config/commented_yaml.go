package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// marshalWithComments builds a yaml.Node tree from a config value with comments from struct tags
// Returns the root content node (encoder will wrap in document)
func marshalWithComments(v interface{}) (*yaml.Node, error) {
	return buildNode(reflect.ValueOf(v), reflect.TypeOf(v))
}

// buildNode recursively builds a yaml.Node from a reflect.Value
func buildNode(val reflect.Value, typ reflect.Type) (*yaml.Node, error) {
	// Handle nil pointers and interfaces
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!null",
				Value: "null",
			}, nil
		}
		val = val.Elem()
		typ = val.Type()
	}

	switch val.Kind() {
	case reflect.Struct:
		return buildStructNode(val, typ)
	case reflect.Slice:
		return buildSliceNode(val, typ)
	case reflect.Map:
		return buildMapNode(val, typ)
	case reflect.String, reflect.Int, reflect.Int32, reflect.Int64, reflect.Bool, reflect.Float64:
		return buildScalarNode(val)
	default:
		// For unknown types, fall back to standard marshalling
		var node yaml.Node
		if err := node.Encode(val.Interface()); err != nil {
			return nil, fmt.Errorf("failed to encode value: %w", err)
		}
		return &node, nil
	}
}

// buildStructNode builds a mapping node from a struct
func buildStructNode(val reflect.Value, typ reflect.Type) (*yaml.Node, error) {
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{},
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanInterface() {
			continue
		}

		// Get yaml tag
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Parse yaml tag (handle omitempty, etc.)
		yamlName := strings.Split(yamlTag, ",")[0]
		if yamlName == "" {
			yamlName = field.Name
		}

		// Check omitempty
		hasOmitempty := strings.Contains(yamlTag, "omitempty")
		if hasOmitempty && isEmptyValue(fieldVal) {
			// For default config, include empty slices/arrays only for top-level fields
			// that are meant to be shown as examples (charts, sources)
			// Skip other empty omitempty fields
			if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
				// Only include if it's charts or sources (top-level collections)
				if yamlName != "charts" && yamlName != "sources" {
					continue
				}
			} else {
				continue
			}
		}

		// Get comment from struct tag
		comment := field.Tag.Get("comment")
		doc := field.Tag.Get("doc")

		// Build key node
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: yamlName,
		}

		// Attach comment to key node (HeadComment appears above the key)
		if comment != "" {
			keyNode.HeadComment = comment
		}
		if doc != "" {
			// For multi-line docs, use HeadComment with newlines
			if comment != "" {
				keyNode.HeadComment = comment + "\n" + doc
			} else {
				keyNode.HeadComment = doc
			}
		}

		// Build value node
		valueNode, err := buildNode(fieldVal, field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to build node for field %s: %w", field.Name, err)
		}

		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node, nil
}

// buildSliceNode builds a sequence node from a slice
func buildSliceNode(val reflect.Value, typ reflect.Type) (*yaml.Node, error) {
	node := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: []*yaml.Node{},
	}

	elemType := typ.Elem()
	for i := 0; i < val.Len(); i++ {
		elemVal := val.Index(i)
		elemNode, err := buildNode(elemVal, elemType)
		if err != nil {
			return nil, fmt.Errorf("failed to build slice element: %w", err)
		}
		node.Content = append(node.Content, elemNode)
	}

	return node, nil
}

// buildMapNode builds a mapping node from a map
func buildMapNode(val reflect.Value, typ reflect.Type) (*yaml.Node, error) {
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{},
	}

	keys := val.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i].Interface()) < fmt.Sprintf("%v", keys[j].Interface())
	})

	for _, k := range keys {
		keyStr := fmt.Sprintf("%v", k.Interface())
		valueVal := val.MapIndex(k)

		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: keyStr,
		}

		valueNode, err := buildNode(valueVal, valueVal.Type())
		if err != nil {
			return nil, fmt.Errorf("failed to build map value: %w", err)
		}

		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node, nil
}

// buildScalarNode builds a scalar node from a primitive value
func buildScalarNode(val reflect.Value) (*yaml.Node, error) {
	node := &yaml.Node{
		Kind: yaml.ScalarNode,
	}

	switch val.Kind() {
	case reflect.String:
		node.Tag = "!!str"
		node.Value = val.String()
	case reflect.Int, reflect.Int32, reflect.Int64:
		node.Tag = "!!int"
		node.Value = fmt.Sprintf("%d", val.Int())
	case reflect.Bool:
		node.Tag = "!!bool"
		if val.Bool() {
			node.Value = "true"
		} else {
			node.Value = "false"
		}
	case reflect.Float64:
		node.Tag = "!!float"
		node.Value = fmt.Sprintf("%g", val.Float())
	default:
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("%v", val.Interface())
	}

	return node, nil
}

// isEmptyValue checks if a value is empty (zero value)
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		// For structs, check if all fields are empty
		for i := 0; i < v.NumField(); i++ {
			if !isEmptyValue(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}
