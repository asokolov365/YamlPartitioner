// Copyright 2023-2024 Andrew Sokolov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package partitioner

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

func newShard(name string, cfg *Config) *shard {
	return &shard{
		name: name,
		cfg:  cfg,
	}
}

type shard struct {
	ctx              context.Context
	anchors          map[string]int
	visitedPaths     map[string]struct{}
	cfg              *Config
	headNode         *yaml.Node
	name             string
	itemsCountBefore int
	itemsCountAfter  int
}

// Reset sets the shard to its initial state.
func (sh *shard) Reset() {
	sh.ctx = nil
	sh.headNode = nil
	sh.itemsCountBefore = 0
	sh.itemsCountAfter = 0
	sh.anchors = make(map[string]int, 100)
	sh.visitedPaths = make(map[string]struct{}, 100)
}

// Run decodes input yaml into a partitioned tree of yaml Nodes.
// When it's done it encodes the partitioned tree of yaml Nodes
// back to yaml format with configured shard io.Writer.
func (sh *shard) Run(ctx context.Context, input []byte, output io.Writer) error {
	var err error
	// reset before sharding
	sh.Reset()
	// it's wrong to store context in a struct itself but
	// this is the only way to pass the context through.
	sh.ctx = ctx

	// Decode input yaml into a partitioned tree of yaml Nodes.
	// This calls shard.UnmarshalYAML()
	if err = yaml.Unmarshal(input, sh); err != nil {
		return fmt.Errorf("failed to unmarshal yaml for %s: %w", sh.name, err)
	}

	// Encode the partitioned tree of yaml Nodes back to yaml format
	// with configured shard writer
	yamlEncoder := yaml.NewEncoder(output)
	yamlEncoder.SetIndent(sh.cfg.resultYamlIndent)

	defer yamlEncoder.Close()

	if err := yamlEncoder.Encode(sh.headNode); err != nil {
		return fmt.Errorf("failed to marshal yaml for %s: %w", sh.name, err)
	}

	return nil
}

// UnmarshalYAML is the unmarshaler that will be called by the YAML processor.
// This dives into the yaml Nodes tree recursively with descendRecursively()
// down to the split point.
// At the split point node it performs consistentHashingFn for node content,
// removing nodes that are not related to this shard.
func (sh *shard) UnmarshalYAML(value *yaml.Node) error {
	if sh.ctx == nil {
		panic("BUG: shard context is not initialized")
	}

	if err := sh.descendRecursively(sh.ctx, value, []string{}); err != nil {
		return err
	}

	if _, ok := sh.visitedPaths[sh.cfg.splitPoint.String()]; !ok {
		return fmt.Errorf("split point path %q not found", sh.cfg.splitPoint)
	}

	sh.headNode = value

	return nil
}

func (sh *shard) descendRecursively(ctx context.Context, node *yaml.Node, currPath []string) error {
	// Checking context before each dive
	select {
	case <-ctx.Done():
		return fmt.Errorf("%s canceled: %w", sh.name, ctx.Err()) // error somewhere, terminate
	default: // default is a must to avoid blocking
	}

	defer sh.markVisited(currPath)

	atSplitPoint := false

	switch sh.whereAt(currPath) {
	case 0:
		atSplitPoint = true
	case 1:
		return nil
	}

	switch node.Kind { //nolint
	case yaml.SequenceNode, yaml.MappingNode:
		step := 1

		if node.Kind == yaml.MappingNode {
			// step is 2 because yaml.MappingNode item is a kv pair
			step = 2
		}

		if atSplitPoint {
			itemsCountBefore := len(node.Content) / step

			if len(node.Anchor) > 0 { // SplitPoint is AnchorNode
				// Store the number of items in the AnchorNode
				sh.anchors[node.Anchor] = itemsCountBefore
			}

			// Increment total items count before partitioning
			sh.itemsCountBefore += itemsCountBefore

			newContent, err := sh.partitionNode(node.Kind, node.Content)
			if err != nil {
				return err
			}

			node.Content = newContent
			sh.itemsCountAfter += len(newContent) / step

			return nil
		}

		for i := 0; i < len(node.Content); i += step {
			var (
				key, item *yaml.Node
				pathElem  string
			)

			if node.Kind == yaml.MappingNode {
				key = node.Content[i]
				item = node.Content[i+1]
				pathElem = key.Value
			} else {
				item = node.Content[i]
				pathElem = "*"
			}

			if err := sh.descendRecursively(ctx, item, append(currPath, pathElem)); err != nil {
				return err
			}
		}

	case yaml.AliasNode:
		if atSplitPoint {
			// Increment total items count before partitioning
			sh.itemsCountBefore += sh.anchors[node.Value]

			var step int

			switch node.Alias.Kind { //nolint
			case yaml.SequenceNode:
				step = 1
			case yaml.MappingNode:
				// step is 2 because yaml.MappingNode item is a kv pair
				step = 2
			default:
				return fmt.Errorf("invalid split point path: node at %q is not shardable", sh.cfg.splitPoint)
			}

			// AnchorNode has already been processed,
			// so its Content length is how many items it has after processing.
			sh.itemsCountAfter += len(node.Alias.Content) / step

			return nil
		}

	default:
		if atSplitPoint {
			return fmt.Errorf("invalid split point path: node at %q is not shardable", sh.cfg.splitPoint)
		}
	}

	return nil
}

func (sh *shard) partitionNode(nodeKind yaml.Kind, oldContent []*yaml.Node) ([]*yaml.Node, error) {
	var (
		key, item *yaml.Node
		step      int
	)

	newContent := []*yaml.Node{}

	switch nodeKind { //nolint
	case yaml.SequenceNode:
		step = 1
	case yaml.MappingNode:
		// step is 2 because yaml.MappingNode item is a kv pair
		step = 2
	default:
		return nil, fmt.Errorf("either SequenceNode or MappingNode can be partitioned")
	}

	for i := 0; i < len(oldContent); i += step {
		if nodeKind == yaml.MappingNode {
			key = oldContent[i]
			item = oldContent[i+1]
		} else {
			item = oldContent[i]
		}

		switch {
		case item.Kind == yaml.AliasNode:
			// ok means the AnchorNode for this AliasNode has already been processed
			if match, ok := sh.anchors[item.Value]; ok {
				// match > 0 means the yaml.Node belongs to this shard
				if match > 0 {
					if nodeKind == yaml.MappingNode {
						newContent = append(newContent, key, item)
					} else {
						newContent = append(newContent, item)
					}
				}
			}

			continue

		case len(item.Anchor) > 0: // AnchorNode
			sh.anchors[item.Anchor] = 0
		}

		itemAsBytes, err := yaml.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal %v: %w", item, err)
		}

		nodeNames := sh.cfg.consistentHashing.GetN(itemAsBytes, sh.cfg.replicasCount)
		if _, ok := nodeNames[sh.name]; ok {
			if nodeKind == yaml.MappingNode {
				newContent = append(newContent, key, item)
			} else {
				newContent = append(newContent, item)
			}

			if len(item.Anchor) > 0 {
				// mark AnchorNode as belonging to this shard
				sh.anchors[item.Anchor] = step // either 1 or 2
			}
		}
	}

	return newContent, nil
}

func (sh *shard) markVisited(path []string) {
	sh.visitedPaths[strings.Join(path, ".")] = struct{}{}
}

func (sh *shard) whereAt(path []string) int {
	maxDeep := sh.cfg.splitPoint.Len()
	if len(path) > maxDeep {
		return 1
	}

	for i := 0; i < len(path); i++ {
		spElem, _ := sh.cfg.splitPoint.Elem(i)
		if path[i] != spElem {
			return 1
		}
	}

	if len(path) == maxDeep {
		return 0
	}

	return -1
}
