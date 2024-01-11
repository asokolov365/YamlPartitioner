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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShard_ShardIdOutOfRange(t *testing.T) {
	t.Parallel()

	var err error

	_, err = NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(3),
		WithResultYamlIndent(2),
		WithSplitPoint("*"),
		WithThisShardID(5),
		WithWorkingDirectory(workDir),
	)
	require.ErrorContains(t, err, "this shard id is out of range")
}

func TestShard_ReplicationTooBig(t *testing.T) {
	t.Parallel()

	var err error

	_, err = NewConfig(
		WithConsistentHashing(getConsistentHashing()),
		WithReplicasCount(3),
		WithResultYamlIndent(2),
		WithSplitPoint("groups.*.rules"),
		WithThisShardID(-1),
		WithWorkingDirectory(workDir),
	)
	require.ErrorContains(t, err, "replication factor is too big")
}
