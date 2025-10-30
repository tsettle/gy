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

var debug = flag.Bool("debug", false, "Enable debug mode")

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
		fmt.Println("gy version 0.0.3")
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
	var node yaml.Node
	err = yaml.Unmarshal(input, &node)
	if err != nil {
		panic(err)
	}

	// Extract the target node
	extracted := extractPath(&node, pattern)
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
	var result *yaml.Node
	if useTrim {
		result = extracted
	} else {
		result = wrapInPath(&node, pattern, extracted)
	}

	output, _ := yaml.Marshal(result)
	fmt.Print(string(output))
}

func wrapInPath(root *yaml.Node, pattern string, extracted *yaml.Node) *yaml.Node {
	// Remove leading dot
	if len(pattern) > 0 && pattern[0] == '.' {
		pattern = pattern[1:]
	}

	parts := splitPath(pattern)

	// Build the tree from the bottom up
	var current *yaml.Node = extracted
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// Handle array indexes like "[0]"
		if len(part) > 0 && part[0] == '[' && part[len(part)-1] == ']' {
			// For arrays, create a sequence node
			seqNode := &yaml.Node{
				Kind: yaml.SequenceNode,
			}

			// Parse the index to find where to place our extracted node
			indexStr := part[1 : len(part)-1]
			index, err := strconv.Atoi(indexStr)
			if err == nil {
				// Create empty nodes before the index
				for j := 0; j < index; j++ {
					seqNode.Content = append(seqNode.Content, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: "null",
						Tag:   "!!null",
					})
				}
				// Add our extracted node at the correct position
				seqNode.Content = append(seqNode.Content, current)
			} else {
				// If we can't parse the index, just return the extracted node
				return extracted
			}
			current = seqNode
		} else {
			// Mapping node - wrap in map
			mapNode := &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Value: part,
						Tag:   "!!str",
					},
					current,
				},
			}
			current = mapNode
		}
	}

	return current
}

func extractPath(node *yaml.Node, pattern string) *yaml.Node {
	if len(pattern) > 0 && pattern[0] == '.' {
		pattern = pattern[1:]
	}

	if pattern == "" {
		return node
	}

	parts := splitPath(pattern)
	current := node

	for partIndex := 0; partIndex < len(parts); {
		part := parts[partIndex]
		if part == "" {
			partIndex++
			continue // Skip empty parts
		}
		if current == nil {
			return nil
		}

		processed := false
		switch current.Kind {
		case yaml.DocumentNode:
			if len(current.Content) > 0 {
				current = current.Content[0]
				processed = true
				// Reprocess the same part with the new current node (don't increment partIndex)
			} else {
				return nil
			}
		case yaml.MappingNode:
			found := false
			for i := 0; i < len(current.Content); i += 2 {
				if i+1 < len(current.Content) && current.Content[i].Value == part {
					current = current.Content[i+1]
					found = true
					processed = true
					partIndex++ // Move to next part
					break
				}
			}
			if !found {
				return nil
			}
		case yaml.SequenceNode:
			// Array access - parse "[0]" into integer
			if len(part) > 2 && part[0] == '[' && part[len(part)-1] == ']' {
				indexStr := part[1 : len(part)-1]
				index, err := strconv.Atoi(indexStr)
				if err != nil || index < 0 || index >= len(current.Content) {
					return nil // Invalid index or out of bounds
				}
				current = current.Content[index]
				processed = true
				partIndex++ // Move to next part
			} else {
				return nil
			}
		default:
			return nil
		}

		if !processed {
			// If we didn't process the part (shouldn't happen in normal flow), move to next
			partIndex++
		}
	}

	return current
}

func splitPath(pattern string) []string {
	var parts []string
	start := 0
	inBracket := false

	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '[':
			if !inBracket {
				// Add the part before the bracket if it's not empty
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
				// Only add if there's content between dots
				if i > start {
					parts = append(parts, pattern[start:i])
				}
				start = i + 1
			}
		}
	}

	// Add any remaining part if it's not empty
	if start < len(pattern) {
		parts = append(parts, pattern[start:])
	}

	return parts
}

func listNode(node *yaml.Node, prefix string, maxDepth, currentDepth int) {
	if node == nil || (maxDepth > 0 && currentDepth >= maxDepth) {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			listNode(node.Content[0], prefix, maxDepth, currentDepth)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) {
				keyNode := node.Content[i]
				valueNode := node.Content[i+1]
				fmt.Printf("%s%s\n", prefix, keyNode.Value)
				listNode(valueNode, prefix+"  ", maxDepth, currentDepth+1)
			}
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			fmt.Printf("%s[%d]\n", prefix, i)
			listNode(item, prefix+"  ", maxDepth, currentDepth+1)
		}
	default:
		// Scalar - no children to list
	}
}
