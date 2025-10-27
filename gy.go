// gy - YAML path extractor and lister
// Copyright (c) 2025 Troy Settle
// Licensed under the MIT License
//
// A fast, lightweight tool for extracting and exploring YAML documents.
// Preserves types and structure without YAML→JSON→YAML conversions.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

func main() {
	// Flag declarations
	trim := flag.Bool("trim", false, "Return only the matched node, not full path")
	trimShort := flag.Bool("t", false, "Return only the matched node (short flag)")
	list := flag.Bool("list", false, "List keys/items under the specified path")
	listShort := flag.Bool("l", false, "List keys/items (short flag)")
	depth := flag.Int("depth", 1, "Maximum depth for list (default: 1)")
	version := flag.Bool("V", false, "Show version information")

	flag.Parse()

	if *version {
		fmt.Println("gy version 0.0.1")
		os.Exit(0)
	}

	useTrim := *trim || *trimShort
	useList := *list || *listShort
	maxDepth := *depth

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: gy [--trim|-t] [--list|-l] [--depth N] <pattern> [filename]")
		os.Exit(1)
	}

	pattern := args[0]
	filename := ""
	if len(args) > 1 {
		filename = args[1]
	}

	// Read from file or stdin
	var input []byte
	var err error
	if filename != "" {
		input, err = os.ReadFile(filename)
	} else {
		input, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		panic(err)
	}

	// Parse YAML
	var data interface{}
	yaml.Unmarshal(input, &data)

	// Extract the target node
	extracted := extractPath(data, pattern)
	if extracted == nil {
		fmt.Printf("Path not found: %s\n", pattern)
		os.Exit(1)
	}

	// --list mode
	if useList {
		listNode(extracted, "", maxDepth, 0)
		os.Exit(0)
	}

	// Normal extraction mode
	var result interface{}
	if useTrim {
		result = extracted
	} else {
		result = wrapInPath(data, pattern, extracted)
	}

	output, _ := yaml.Marshal(result)
	fmt.Print(string(output))
}

func wrapInPath(data interface{}, pattern string, extracted interface{}) interface{} {
	// Remove leading dot
	if len(pattern) > 0 && pattern[0] == '.' {
		pattern = pattern[1:]
	}

	parts := splitPath(pattern)

	// Build the tree from the bottom up
	var current interface{} = extracted
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// Handle array indexes like "[0]"
		if len(part) > 0 && part[0] == '[' && part[len(part)-1] == ']' {
			// For now, skip array reconstruction - complex!
			// Just return the extracted node
			return extracted
		} else {
			// Mapping node - wrap in map
			current = map[string]interface{}{
				part: current,
			}
		}
	}

	return current
}

func extractPath(data interface{}, pattern string) interface{} {
	if len(pattern) > 0 && pattern[0] == '.' {
		pattern = pattern[1:]
	}

	parts := splitPath(pattern)
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			// Regular map access
			var exists bool
			current, exists = v[part]
			if !exists {
				return nil
			}
		case []interface{}:
			// Array access - parse "[0]" into integer
			if len(part) > 2 && part[0] == '[' && part[len(part)-1] == ']' {
				indexStr := part[1 : len(part)-1]
				index, err := strconv.Atoi(indexStr)
				if err != nil || index < 0 || index >= len(v) {
					return nil // Invalid index or out of bounds
				}
				current = v[index]
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// Basic path splitter (handles dots, not brackets yet)
func splitPath(pattern string) []string {
	var parts []string
	start := 0
	inBracket := false

	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '[':
			if !inBracket {
				// Add the part before the bracket
				if i > start {
					parts = append(parts, pattern[start:i])
				}
				start = i
				inBracket = true
			}
		case ']':
			if inBracket {
				// Add the bracket part including the brackets
				parts = append(parts, pattern[start:i+1])
				start = i + 1
				inBracket = false
			}
		case '.':
			if !inBracket {
				if i > start {
					parts = append(parts, pattern[start:i])
				}
				start = i + 1
			}
		}
	}

	// Add any remaining part
	if start < len(pattern) {
		parts = append(parts, pattern[start:])
	}
	return parts
}

func listNode(node interface{}, prefix string, maxDepth, currentDepth int) {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return
	}

	switch v := node.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fmt.Printf("%s%s\n", prefix, key)
			listNode(value, prefix+"  ", maxDepth, currentDepth+1)
		}
	case []interface{}:
		for i, item := range v {
			fmt.Printf("%s[%d]\n", prefix, i)
			listNode(item, prefix+"  ", maxDepth, currentDepth+1)
		}
	default:
		// Scalar - no children to list
	}
}
