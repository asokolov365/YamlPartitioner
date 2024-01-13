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
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/asokolov365/YamlPartitioner/lib/hrw"
	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/require"
)

var shardNames = []string{
	"alpha",
	"beta",
	"gamma",
	"delta",
	"epsilon",
}

var workDir = "/tmp"

func getConsistentHashing() ConsistentHashing {
	rndv, _ := hrw.New(xxhash.Sum64, shardNames...)
	return rndv
}

func cleanup() error {
	for _, name := range shardNames {
		path := filepath.Join(workDir, name)
		// fmt.Printf("Removing %q\n", path)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %q: %w", path, err)
		}
	}

	return nil
}

func TestRun_SequenceNodePath(t *testing.T) {
	var err error

	// grep 'expr:' testdata/rules/kube-good.yaml | wc -l
	// => 160
	expectedTotalItems := 160
	expectedItemsAfter := map[string]int{
		"alpha":   64,
		"beta":    72,
		"gamma":   66,
		"delta":   64,
		"epsilon": 54,
	}
	expectedMD5Sum := map[string]string{
		"alpha":   "822b25a95710c0f75d48ed3ba818daae",
		"beta":    "c2c34efe88668e57483af492276ee057",
		"gamma":   "23005f1b371b8a18e1682d25c6e7cf36",
		"delta":   "ff8b87d66cf4500c64110ad93a903ad5",
		"epsilon": "93839beeff7e2c33446924594aad39a8",
	}

	inputFile, err := filepath.Abs("../../testdata/rules/kube-good.yaml")
	require.NoError(t, err)

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.rules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	p, err := WithConfig(cfg, inputFile, "")
	require.NoError(t, err)
	err = p.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedTotalItems, p.totalItemsBefore)

	for _, name := range shardNames {
		require.Equal(t, expectedItemsAfter[name], p.shardItemsCount[name])

		resultFile := filepath.Join(workDir, name, p.outputFile)

		f, err := os.Open(resultFile)
		require.NoError(t, err)

		h := md5.New()

		_, err = io.Copy(h, f)
		require.NoError(t, err)

		f.Close()

		require.Equal(t, expectedMD5Sum[name], fmt.Sprintf("%x", h.Sum(nil)))
	}

	err = cleanup()
	require.NoError(t, err)
}

func TestRun_MappingNodePath(t *testing.T) {
	var err error

	// grep 'prober:' testdata/dir/blackbox-good.yml | wc -l
	// => 10
	expectedTotalItems := 10
	expectedItemsAfter := map[string]int{
		"alpha":   6,
		"beta":    7,
		"gamma":   3,
		"delta":   2,
		"epsilon": 2,
	}
	expectedMD5Sum := map[string]string{
		"alpha":   "dc6693b15498c2eac18a704c31f3e79c",
		"beta":    "372e66a33b7181dc9c0b096315628b59",
		"gamma":   "04520bbde057ab965291a5d4d12817fc",
		"delta":   "39a249b9c9057b394dd0376fe4ea7850",
		"epsilon": "e34be287ee5b51f31d410a0aa41c2005",
	}

	inputFile, err := filepath.Abs("../../testdata/dir/blackbox-good.yml")
	require.NoError(t, err)

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("modules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	p, err := WithConfig(cfg, inputFile, "")
	require.NoError(t, err)

	err = p.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedTotalItems, p.totalItemsBefore)

	for _, name := range shardNames {
		require.Equal(t, expectedItemsAfter[name], p.shardItemsCount[name])

		resultFile := filepath.Join(workDir, name, p.outputFile)

		f, err := os.Open(resultFile)
		require.NoError(t, err)

		h := md5.New()
		_, err = io.Copy(h, f)
		require.NoError(t, err)

		f.Close()

		require.Equal(t, expectedMD5Sum[name], fmt.Sprintf("%x", h.Sum(nil)))
	}

	err = cleanup()
	require.NoError(t, err)
}

func TestRun_InputFileNotFound(t *testing.T) {
	t.Parallel()

	var err error

	inputFile, err := filepath.Abs("../../testdata/dir/blackbox.yml")
	require.NoError(t, err)

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("modules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	p, err := WithConfig(cfg, inputFile, "")
	require.NoError(t, err)

	err = p.Run(context.Background())
	require.ErrorContains(t, err, "failed to read input file")

	err = cleanup()
	require.NoError(t, err)
}

func TestRun_InvalidCommonPrefix(t *testing.T) {
	t.Parallel()

	var err error

	inputFile, err := filepath.Abs("../../testdata/dir/blackbox.yml")
	require.NoError(t, err)

	cfg, err := NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(2),
		WithResultYamlIndent(2),
		WithSplitPoint("modules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.NoError(t, err)

	_, err = WithConfig(cfg, inputFile, "testdata/rules")
	require.ErrorContains(t, err, "invalid common prefix for ")
}
