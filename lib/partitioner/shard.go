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

	"github.com/asokolov365/YamlPartitioner/lib/bytesutil"

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
	visitedPaths     map[string]struct{}
	cfg              *Config
	headNode         *yaml.Node
	name             string
	itemsCountBefore int
	itemsCountAfter  int
}

// Reset sets the shard to its initial state
func (sh *shard) Reset() {
	sh.ctx = nil
	sh.headNode = nil
	sh.itemsCountBefore = 0
	sh.itemsCountAfter = 0
	sh.visitedPaths = map[string]struct{}{}
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
		return fmt.Errorf("error unmarshal yaml for %s: %w", sh.name, err)
	}

	// Encode the partitioned tree of yaml Nodes back to yaml format
	// with configured shard writer
	yamlEncoder := yaml.NewEncoder(output)
	yamlEncoder.SetIndent(sh.cfg.resultYamlIndent)
	defer yamlEncoder.Close()

	if err := yamlEncoder.Encode(sh.headNode); err != nil {
		return fmt.Errorf("error marshal yaml for %s: %w", sh.name, err)
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
		// fmt.Printf("%s canceled\n", sh.name)
		return ctx.Err() // error somewhere, terminate
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

	switch node.Kind {
	case yaml.SequenceNode:
		if atSplitPoint {
			sh.itemsCountBefore += len(node.Content)
			newContent := []*yaml.Node{}

			for _, item := range node.Content {
				valueAsBytes, err := yaml.Marshal(item)
				if err != nil {
					return err
				}
				nodeNames := sh.cfg.consistentHashing.GetN(valueAsBytes, sh.cfg.replicasCount)
				if _, ok := nodeNames[sh.name]; ok {
					newContent = append(newContent, item)
				}
			}
			node.Content = newContent
			sh.itemsCountAfter += len(newContent)
			return nil
		}

		for _, item := range node.Content {
			if err := sh.descendRecursively(ctx, item, append(currPath, "*")); err != nil {
				return err
			}
		}

	case yaml.MappingNode:
		var key, value *yaml.Node
		if atSplitPoint {
			// divide by 2 because yaml.MappingNode item is key and value pair
			sh.itemsCountBefore += len(node.Content) / 2
			newContent := []*yaml.Node{}

			for i := 0; i < len(node.Content); i += 2 {
				key = node.Content[i]
				value = node.Content[i+1]

				nodeNames := sh.cfg.consistentHashing.GetN(bytesutil.ToUnsafeBytes(key.Value), sh.cfg.replicasCount)
				if _, ok := nodeNames[sh.name]; ok {
					newContent = append(newContent, key, value)
				}
			}
			node.Content = newContent
			sh.itemsCountAfter += len(newContent) / 2
			return nil
		}

		for i := 0; i < len(node.Content); i += 2 {
			key = node.Content[i]
			value = node.Content[i+1]
			if err := sh.descendRecursively(ctx, value, append(currPath, key.Value)); err != nil {
				return err
			}
		}
	case yaml.AliasNode:
		// Always pass AliasNodes as is
		return nil
	default:
		if atSplitPoint {
			return fmt.Errorf("invalid split point path: node at %q is not shardable", sh.cfg.splitPoint)
		}
	}

	return nil
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
