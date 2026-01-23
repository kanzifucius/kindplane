package helm

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadValuesFile loads a YAML values file and returns it as a map
func LoadValuesFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file %s: %w", path, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file %s: %w", path, err)
	}

	return values, nil
}

// MergeValues merges multiple values maps, with later maps taking precedence
// The order is: valuesFiles (in order), then inline values
func MergeValues(valuesFiles []string, inlineValues map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Load and merge values files in order
	for _, path := range valuesFiles {
		fileValues, err := LoadValuesFile(path)
		if err != nil {
			return nil, err
		}
		result = mergeMaps(result, fileValues)
	}

	// Merge inline values last (highest priority)
	if inlineValues != nil {
		result = mergeMaps(result, inlineValues)
	}

	return result, nil
}

// mergeMaps recursively merges two maps, with src taking precedence over dst
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}

	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			// Both have the key, check if both are maps for recursive merge
			srcMap, srcIsMap := srcVal.(map[string]interface{})
			dstMap, dstIsMap := dstVal.(map[string]interface{})

			if srcIsMap && dstIsMap {
				// Both are maps, merge recursively
				dst[key] = mergeMaps(dstMap, srcMap)
			} else {
				// Not both maps, src wins
				dst[key] = srcVal
			}
		} else {
			// Key doesn't exist in dst, just set it
			dst[key] = srcVal
		}
	}

	return dst
}

// ParseSetValues parses --set style key=value pairs into a map
// Supports nested keys like "controller.replicaCount=2"
func ParseSetValues(setValues []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, setValue := range setValues {
		// Find the first = sign
		eqIdx := -1
		for i, c := range setValue {
			if c == '=' {
				eqIdx = i
				break
			}
		}

		if eqIdx == -1 {
			return nil, fmt.Errorf("invalid --set value: %s (expected key=value)", setValue)
		}

		key := setValue[:eqIdx]
		value := setValue[eqIdx+1:]

		if key == "" {
			return nil, fmt.Errorf("invalid --set value: %s (empty key)", setValue)
		}

		// Set the nested value
		if err := setNestedValue(result, key, value); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// setNestedValue sets a value in a nested map using dot notation
// e.g., "controller.replicaCount" -> result["controller"]["replicaCount"]
func setNestedValue(m map[string]interface{}, key string, value interface{}) error {
	keys := splitKey(key)
	current := m

	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key, set the value
			current[k] = value
		} else {
			// Not the last key, ensure intermediate map exists
			if _, exists := current[k]; !exists {
				current[k] = make(map[string]interface{})
			}

			// Check if it's a map
			nextMap, ok := current[k].(map[string]interface{})
			if !ok {
				return fmt.Errorf("cannot set nested value: %s is not a map", k)
			}
			current = nextMap
		}
	}

	return nil
}

// splitKey splits a dot-notation key into parts
// Handles escaped dots with backslash
func splitKey(key string) []string {
	var parts []string
	var current string
	escaped := false

	for _, c := range key {
		if escaped {
			current += string(c)
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			continue
		}

		current += string(c)
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
