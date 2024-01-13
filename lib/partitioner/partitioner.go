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

// Package partitioner implements utility routines for partitioning
// of a single YAML file.
package partitioner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// New creates a new partitioner for the given inputFile.
// inputFile is an absolute path to a file that needs to be partitioned.
//
// commonPath:
// Say inputFile is <commonPath>/<path-to-file.yml>,
// the result of partitioning will be written to
// <workDir>/<shardName>/<path-to-file.yml>.
func New(inputFile, commonPath string, opts ...Option) (*Partitioner, error) {
	cfg, err := NewConfig(opts...)
	if err != nil {
		return nil, err
	}

	return WithConfig(cfg, inputFile, commonPath)
}

// WithConfig is being used to create a new partitioner
// with the previously created configuration.
func WithConfig(cfg *Config, inputFile, commonPath string) (*Partitioner, error) {
	if len(commonPath) == 0 {
		commonPath = filepath.Dir(inputFile)
	}

	p := &Partitioner{
		inputFile: inputFile,
		cfg:       cfg,
		// shardItemsCount: make(map[string]int, cfg.NodesCount()),
	}

	outputFile, ok := strings.CutPrefix(inputFile, commonPath)
	if !ok || len(outputFile) == 0 {
		return nil, fmt.Errorf("invalid common prefix for %q: %q", inputFile, commonPath)
	}

	p.outputFile = outputFile

	return p, nil
}

// Partitioner represents the structure for partitioning
// a given input file.
type Partitioner struct {
	cfg              *Config
	shardItemsCount  map[string]int
	inputFile        string
	outputFile       string
	report           string
	totalItemsBefore int
	mu               sync.Mutex
}

// Report returns a partitioning report.
func (p *Partitioner) Report() string {
	return p.report
}

// ShardItemsCount returns how many items got each shard.
func (p *Partitioner) ShardItemsCount() map[string]int {
	return p.shardItemsCount
}

// Reset sets the partitioner to its initial state.
func (p *Partitioner) Reset() {
	p.totalItemsBefore = 0
	p.shardItemsCount = make(map[string]int, p.cfg.NodesCount())
}

// Run performs the partitioning of a given input file
// accordingly to settings in the Partitioner Config.
func (p *Partitioner) Run(ctx context.Context) error {
	p.Reset()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel() // releases resources if slow operation completes before timeout elapses

	input, err := os.ReadFile(p.inputFile)
	if err != nil {
		p.cleanupOnError()
		return fmt.Errorf("failed to read input file: %w", err)
	}

	startTime := time.Now()

	// any error breaks the entire group but not the other partitioners.
	g, gCtx := errgroup.WithContext(ctx)

	shards := make([]*shard, 0, p.cfg.NodesCount())

	for i, name := range p.cfg.NodeNames() {
		// Skipping partitioning if thisShardID has set
		if p.cfg.thisShardID >= 0 && p.cfg.thisShardID != i {
			continue
		}

		// Checking if context canceled before running a shard
		select {
		case <-ctx.Done():
			p.cleanupOnError()
			return fmt.Errorf("%s canceled: %w", name, ctx.Err()) // error somewhere, terminate
		default: // default is a must to avoid blocking
		}

		shardName := name
		shard := newShard(shardName, p.cfg)
		shards = append(shards, shard)

		g.Go(func() error {
			// TODO: handle writing to stdout with bytes.Buffer
			outputFile := filepath.Join(p.cfg.workDir, shardName, p.outputFile)
			f, err := createOutputFile(outputFile)
			if err != nil {
				return err
			}

			if err := shard.Run(gCtx, input, f); err != nil {
				f.Close()
				os.Remove(outputFile)
				return err
			}

			if err := p.setItemsBefore(shard.itemsCountBefore); err != nil {
				f.Close()
				os.Remove(outputFile)
				return err
			}

			f.Close()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		p.cleanupOnError()
		return fmt.Errorf("failed to partition %q: %w", p.inputFile, err)
	}

	finishTime := time.Since(startTime)

	// Write partitioning stats
	var report strings.Builder

	report.WriteString(
		fmt.Sprintf("Partitioning %q of size %d bytes finished in %d ms\n",
			p.outputFile, len(input), finishTime.Milliseconds()),
	)

	report.WriteString(
		fmt.Sprintf("Found %d items at path %q, partitioned them into %d shards with RF=%d\n",
			p.totalItemsBefore, p.cfg.splitPoint, p.cfg.NodesCount(), p.cfg.replicasCount),
	)

	for _, shard := range shards {
		if err := p.setItemsBefore(shard.itemsCountBefore); err != nil {
			p.cleanupOnError()
			return err
		}

		p.shardItemsCount[shard.name] = shard.itemsCountAfter

		if shard.itemsCountAfter == 0 {
			outputFile := filepath.Join(p.cfg.workDir, shard.name, p.outputFile)
			os.Remove(outputFile)

			report.WriteString(
				fmt.Sprintf("Shard %q got %d items in resulting yaml (output file is not created)\n",
					shard.name, shard.itemsCountAfter),
			)
		} else {
			report.WriteString(
				fmt.Sprintf("Shard %q got %d items in resulting yaml\n",
					shard.name, shard.itemsCountAfter),
			)
		}
	}

	p.report = report.String()

	return nil
}

func (p *Partitioner) setItemsBefore(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.totalItemsBefore == 0 {
		p.totalItemsBefore = n

		return nil
	}

	if p.totalItemsBefore != n {
		return fmt.Errorf("items consistency error")
	}

	return nil
}

func (p *Partitioner) cleanupOnError() {
	p.report = fmt.Sprintf("Partitioning %q failed. See errors report.\n", p.outputFile)

	for i, name := range p.cfg.NodeNames() {
		// Skipping other shards if thisShardID has set
		if p.cfg.thisShardID >= 0 && p.cfg.thisShardID != i {
			continue
		}

		outputFile := filepath.Join(p.cfg.workDir, name, p.outputFile)
		os.Remove(outputFile)
	}
}

func createOutputFile(path string) (*os.File, error) {
	fileDir := filepath.Dir(path)

	if err := os.MkdirAll(fileDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to make directory %q: %w", fileDir, err)
	}

	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o644) //nolint
}
