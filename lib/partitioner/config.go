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
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ConsistentHashing represent an abstract interface for consistent hashing.
type ConsistentHashing interface {
	// NodeNames returns the list of node names in the ConsistentHashing
	NodeNames() []string
	// NodesCount returns the number of nodes in the ConsistentHashing
	NodesCount() int
	// Get gets the most suitable node name for a key
	Get(key []byte) string
	// GetN gets N most suitable node names for a key
	GetN(key []byte, replicasCount int) map[string]struct{}
}

// Config defines common configuration for yaml partitioning.
// Config must be immutable.
type Config struct {
	consistentHashing ConsistentHashing
	splitPoint        *splitPoint
	workDir           string
	thisShardID       int
	replicasCount     int
	resultYamlIndent  int
}

// NodesCount returns the number of nodes in the ConsistentHashing.
func (c *Config) NodesCount() int {
	return c.consistentHashing.NodesCount()
}

// NodeNames returns names of nodes in the ConsistentHashing.
func (c *Config) NodeNames() []string {
	return c.consistentHashing.NodeNames()
}

// WorkDir returns the working directory (temp dir).
func (c *Config) WorkDir() string {
	return c.workDir
}

// NewConfig creates a new common configuration for yaml partitioning.
//
// Example:
//
//	var shardNames = []string{
//		"alpha",
//		"beta",
//		"gamma",
//		"delta",
//		"epsilon",
//	}
//
// rndv, _ := hrw.New(xxhash.Sum64, shardNames...)
//
// cfg, err := NewConfig(
//
//	WithConsistentHashing(rndv),
//	WithReplicasCount(1),
//	WithResultYamlIndent(2),
//	WithSplitPoint("groups.*.rules"),
//	WithThisShardID(-1),
//
// )
// .
func NewConfig(opts ...Option) (*Config, error) {
	cfg := &Config{
		replicasCount:    1,
		resultYamlIndent: 2,
		thisShardID:      -1,
		workDir:          os.TempDir(),
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Checking REQUIRED settings
	if cfg.consistentHashing == nil {
		return nil, fmt.Errorf("consistent hashing is not set")
	}

	if cfg.splitPoint == nil || len(cfg.splitPoint.slice) == 0 || len(cfg.splitPoint.str) == 0 {
		return nil, fmt.Errorf("split point path is not set")
	}

	shardsCount := cfg.consistentHashing.NodesCount()

	if shardsCount < 2 {
		return nil, fmt.Errorf("number of nodes in consistent hashing must be >=2")
	}

	if cfg.thisShardID >= shardsCount {
		return nil, fmt.Errorf("this shard id is out of range")
	}

	if cfg.replicasCount > (shardsCount / 2) {
		return nil, fmt.Errorf("replication factor is too big")
	}

	return cfg, nil
}

// Option is a type for a function that accepts a pointer to
// an empty or minimal Config struct created in the Config constructor.
// Option represents Functional Options Pattern.
// See this article for details -
// https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis .
type Option func(*Config) error

// WithConsistentHashing sets the field with the object that
// implements ConsistentHashing interface
// REQUIRED .
func WithConsistentHashing(h ConsistentHashing) Option {
	if h.NodesCount() < 2 {
		return func(c *Config) error {
			return fmt.Errorf("number of nodes must be >= 2")
		}
	}

	return func(c *Config) error {
		c.consistentHashing = h
		return nil
	}
}

// WithSplitPoint sets the path to a yaml Node, which represents the
// so-called SplitPoint.
// YamlPartitioner dives into the YAML structure up to the given yaml Node
// and starts sharding items from there.
// Note: the SplitPoint yaml Node Kind must be either a SequenceNode (list)
// or a MappingNode (map).
// The SplitPoint must be in format "<key>", "<key>.*.<key>"
// REQUIRED .
func WithSplitPoint(s string) Option {
	sp, err := newSplitPoint(s)
	if err != nil {
		return func(c *Config) error { return err }
	}

	return func(c *Config) error {
		c.splitPoint = sp
		return nil
	}
}

// WithThisShardID sets the shard id for which YamlPartitioner
// creates the resulting YAML(s).
// This defaults to -1, meaning that YamlPartitioner
// creates the resulting YAML for all shards.
func WithThisShardID(id int) Option {
	return func(c *Config) error {
		c.thisShardID = id
		return nil
	}
}

// WithReplicasCount sets the number of replicas.
// Meaning how many shards will get the same data.
// This defaults to 1.
func WithReplicasCount(n int) Option {
	if n < 1 {
		n = 1
	}

	return func(c *Config) error {
		c.replicasCount = n
		return nil
	}
}

// WithResultYamlIndent sets the indentation used for Yaml encoding.
// This defaults to 2.
func WithResultYamlIndent(i int) Option {
	if i < 2 || i > 9 {
		i = 2
	}

	return func(c *Config) error {
		c.resultYamlIndent = i
		return nil
	}
}

// WithWorkingDirectory sets the directory where.
// This defaults to os.TempDir() .
func WithWorkingDirectory(path string) Option {
	// TODO: handle writing to stdout with bytes.Buffer
	// if path == "stdout" {
	// 	return func(c *Config) error {
	// 		c.workDir = path
	// 		return nil
	// 	}
	// }
	path, err := filepath.Abs(path)
	if err != nil {
		return func(c *Config) error {
			return fmt.Errorf("error getting absolute path for %q: %w", path, err)
		}
	}

	if err := validateWorkDir(path); err != nil {
		return func(c *Config) error { return err }
	}

	return func(c *Config) error {
		c.workDir = path
		return nil
	}
}

func validateWorkDir(path string) error {
	fileInfo, err := os.Stat(path)

	switch {
	case err == nil: // path exists
		// path is a directory
		if fileInfo.IsDir() {
			return nil
		}
		// path is a file
		return fmt.Errorf("file is an existing file: %q", path)

	case errors.Is(err, os.ErrNotExist):
		// path does *not* exist
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("error making directory %q: %w", path, err)
		}

		return nil

	default:
		// Schrodinger: path may or may not exist. See err for details.
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for path existence
		return fmt.Errorf("schrodinger: %q may or may not exist: %w", path, err)
	}
}
