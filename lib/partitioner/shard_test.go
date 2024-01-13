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
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Case where the split point is a yaml.SequenceNode that contains AnchorNodes and AliasNodes.
func TestShard_YamlWithAnchors1(t *testing.T) {
	t.Parallel()

	var err error

	// See ../../testdata/anchors/case1.yml: 10 rules + 2 * 10 aliases
	expectedTotalItems := 30
	expectedItemsAfter := map[string]int{
		"alpha":   9,
		"beta":    12,
		"gamma":   12,
		"delta":   12,
		"epsilon": 15,
	}

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.rules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/anchors/case1.yml")
	require.NoError(t, err)

	for _, name := range shardNames {
		var buf bytes.Buffer

		shard := newShard(name, cfg)

		err = shard.Run(context.Background(), input, &buf)
		require.NoError(t, err)

		// fmt.Println(buf.String())

		require.Equal(t, expectedTotalItems, shard.itemsCountBefore, "Shard: %s", name)
		require.Equal(t, expectedItemsAfter[name], shard.itemsCountAfter, "Shard: %s", name)
	}
}

// Case where the split point is a yaml.SequenceNode with Anchor or AliasNode.
func TestShard_YamlWithAnchors2(t *testing.T) {
	t.Parallel()

	var err error

	// See ../../testdata/anchors/case2.yml: 10 rules + 2 aliases to rules
	expectedTotalItems := 30
	expectedItemsAfter := map[string]int{
		"alpha":   6,
		"beta":    3,
		"gamma":   9,
		"delta":   6,
		"epsilon": 6,
	}

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(1),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.rules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/anchors/case2.yml")
	require.NoError(t, err)

	for _, name := range shardNames {
		var buf bytes.Buffer

		shard := newShard(name, cfg)

		err = shard.Run(context.Background(), input, &buf)
		require.NoError(t, err)

		// fmt.Println(buf.String())

		require.Equal(t, expectedTotalItems, shard.itemsCountBefore, "Shard: %s", name)
		require.Equal(t, expectedItemsAfter[name], shard.itemsCountAfter, "Shard: %s", name)
	}
}

// Case where the split point is a yaml.MappingNode that contains AnchorNodes and AliasNodes.
func TestShard_YamlWithAnchors3(t *testing.T) {
	t.Parallel()

	var err error

	// See ../../testdata/anchors/case3.yml: 10 rules + 2 * 10 aliases
	expectedTotalItems := 30
	expectedItemsAfter := map[string]int{
		"alpha":   18,
		"beta":    9,
		"gamma":   12,
		"delta":   6,
		"epsilon": 15,
	}

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.rules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/anchors/case3.yml")
	require.NoError(t, err)

	for _, name := range shardNames {
		var buf bytes.Buffer

		shard := newShard(name, cfg)

		err = shard.Run(context.Background(), input, &buf)
		require.NoError(t, err)

		// fmt.Println(buf.String())

		require.Equal(t, expectedTotalItems, shard.itemsCountBefore, "Shard: %s", name)
		require.Equal(t, expectedItemsAfter[name], shard.itemsCountAfter, "Shard: %s", name)
	}
}

// Case where the split point is a yaml.MappingNode with Anchor or AliasNode.
func TestShard_YamlWithAnchors4(t *testing.T) {
	t.Parallel()

	var err error

	// See ../../testdata/anchors/case4.yml: 10 rules + 2 * 10 aliases
	expectedTotalItems := 30
	expectedItemsAfter := map[string]int{
		"alpha":   3,
		"beta":    12,
		"gamma":   6,
		"delta":   3,
		"epsilon": 6,
	}

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(1),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.rules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/anchors/case4.yml")
	require.NoError(t, err)

	for _, name := range shardNames {
		var buf bytes.Buffer

		shard := newShard(name, cfg)

		err = shard.Run(context.Background(), input, &buf)
		require.NoError(t, err)

		// fmt.Println(buf.String())

		require.Equal(t, expectedTotalItems, shard.itemsCountBefore, "Shard: %s", name)
		require.Equal(t, expectedItemsAfter[name], shard.itemsCountAfter, "Shard: %s", name)
	}
}

func TestShard_SplitPointPathNonShardable(t *testing.T) {
	t.Parallel()

	var err error

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.name"),
		WithThisShardID(0),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/anchors/case1.yml")
	require.NoError(t, err)

	var buf bytes.Buffer

	shard := newShard(shardNames[0], cfg)

	err = shard.Run(context.Background(), input, &buf)
	require.ErrorContains(t, err, "is not shardable")
}

func TestShard_InvalidSplitPointPath(t *testing.T) {
	t.Parallel()

	var err error

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.nonexisting"),
		WithThisShardID(0),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/anchors/case1.yml")
	require.NoError(t, err)

	var buf bytes.Buffer

	shard := newShard(shardNames[0], cfg)

	err = shard.Run(context.Background(), input, &buf)
	require.ErrorContains(t, err, "split point path \"groups.*.nonexisting\" not found")
}

func TestShard_NotYaml(t *testing.T) {
	t.Parallel()

	var err error

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("modules"),
		WithThisShardID(0),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	input, err := os.ReadFile("../../testdata/rfc0822.txt")
	require.NoError(t, err)

	shard := newShard(shardNames[0], cfg)
	err = shard.Run(context.Background(), input, io.Discard)
	require.ErrorContains(t, err, "failed to unmarshal yaml for ")
}
