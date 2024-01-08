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

package app

import (
	"fmt"

	"github.com/asokolov365/YamlPartitioner/lib/hrw"
	"github.com/asokolov365/YamlPartitioner/lib/partitioner"
	"github.com/cespare/xxhash/v2"
)

var (
	// MainConfig represents the main configuration.
	// It's being used with snakecharmer as a ResultStruct
	// for generating app flags, config params, etc.
	MainConfig        *Config
	consistentHashing partitioner.ConsistentHashing
)

// InitConfig creates a new Config with the default values.
// SnakeCharmer will override the values with params
// from the config file, ENV vars, or flags
func InitConfig() {
	splitPointPath := "*"
	srcFilePath := "./**/*.{yml,yaml}"
	dstDirPath := "/tmp"
	shardBaseName := "instance"
	shardsNumber := 0
	shardID := -1
	replicationFactor := 1
	MainConfig = &Config{
		SplitPointPath:    &splitPointPath,
		SrcFilePath:       &srcFilePath,
		DstDirPath:        &dstDirPath,
		ShardBaseName:     &shardBaseName,
		ShardsNumber:      &shardsNumber,
		ShardID:           &shardID,
		ReplicationFactor: &replicationFactor,
	}
}

// Config represents the *yp* configuration
type Config struct {
	// Split point path in YAML, e.g. 'groups.*.rules'. This must be a SequenceNode or MappingNode."
	SplitPointPath *string `mapstructure:"split-at,omitempty" usage:"REQUIRED. Split point path in YAML, e.g. 'groups.*.rules'. This must be a YAML SequenceNode or MappingNode." env:"YP_SPLIT_POINT"`
	// Path to input YAML file or directory that needs to be partitioned.
	SrcFilePath *string `mapstructure:"src,omitempty" usage:"REQUIRED. Path to input YAML file or directory that needs to be partitioned." env:"YP_SRC_PATH"`
	// Output directory where partitioned YAML files are stored.
	DstDirPath *string `mapstructure:"dst,omitempty" usage:"Output directory where partitioned YAML files are stored." env:"YP_DST_PATH"`
	// Basename that used for automatic creation of the list of unnamed shards.
	ShardBaseName *string `mapstructure:"shard-basename,omitempty" usage:"Basename that used for automatic creation of the list of shards." env:"YP_SHARD_BASENAME"`
	// How many of unnamed shards to create.
	ShardsNumber *int `mapstructure:"shards-number,omitempty" usage:"How many shards to create." env:"YP_SHARDS_NUMBER"`
	// This shard ID. If not set, *yp* writes content for all shards.
	ShardID *int `mapstructure:"shard-id,omitempty" usage:"This shard ID. This represents the index of this instance in the list of shards. If not set (-1), *yp* writes content for all instances."  env:"YP_SHARD_ID"`
	// Replication Factor. This defines how many shards get the same item.
	ReplicationFactor *int `mapstructure:"replication,omitempty" usage:"Replication Factor. This defines how many shards get the same YAML item." env:"YP_REPLICATION_FACTOR"`
}

// ConsistentHashing generates list of node names and creates
// a new Rendezvous that implements partitioner.ConsistentHashing interface.
func (c *Config) ConsistentHashing() partitioner.ConsistentHashing {
	if consistentHashing != nil {
		return consistentHashing
	}

	shardNames := make([]string, *c.ShardsNumber)
	for i := 0; i < len(shardNames); i++ {
		shardNames[i] = fmt.Sprintf("%s.%d", *c.ShardBaseName, i)
	}

	consistentHashing, _ = hrw.New(xxhash.Sum64, shardNames...)

	return consistentHashing
}
