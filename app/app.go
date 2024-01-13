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

// Package app implements a YAML partitioning application.
package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/asokolov365/YamlPartitioner/lib/filesutil"
	"github.com/asokolov365/YamlPartitioner/lib/partitioner"
)

var mainJob *job

// Init initializes the partitioning job.
func Init() error {
	// inputFiles, err := filesreader.ListFiles(*MainConfig.SrcFilePath)
	inputFiles, err := filesutil.List(*MainConfig.SrcFilePath)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(inputFiles) < 1 {
		return fmt.Errorf("no file(s) found for pattern %q", *MainConfig.SrcFilePath)
	}

	tmpDir, err := os.MkdirTemp(os.TempDir(), "yp.")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	cfg, err := partitioner.NewConfig(
		partitioner.WithConsistentHashing(MainConfig.ConsistentHashing()),
		partitioner.WithReplicasCount(*MainConfig.ReplicationFactor),
		partitioner.WithSplitPoint(*MainConfig.SplitPointPath),
		partitioner.WithThisShardID(*MainConfig.ShardID),
		partitioner.WithWorkingDirectory(tmpDir),
	)
	if err != nil {
		return fmt.Errorf("failed to init partitioner config: %w", err)
	}

	mainJob = &job{
		cfg:          cfg,
		partitioners: make(map[string]*partitioner.Partitioner, len(inputFiles)),
	}

	commonPath := filesutil.LongestCommonPath(inputFiles)

	for _, file := range inputFiles {
		p, err := partitioner.WithConfig(cfg, file, commonPath)
		if err != nil {
			return fmt.Errorf("failed to init partitioner instance: %w", err)
		}

		mainJob.partitioners[file] = p
	}

	return nil
}

type job struct {
	cfg          *partitioner.Config
	partitioners map[string]*partitioner.Partitioner
	mu           sync.Mutex
}

// Run starts the partitioning.
func Run(ctx context.Context, verbose bool) error {
	if mainJob == nil {
		panic("BUG: mainJob is not initialized")
	}

	return mainJob.run(ctx, verbose)
}

func (job *job) run(ctx context.Context, verbose bool) error {
	job.mu.Lock()
	defer job.mu.Unlock()

	if err := os.MkdirAll(*MainConfig.DstDirPath, 0o755); err != nil {
		return fmt.Errorf("failed to make directory %q: %w", *MainConfig.DstDirPath, err)
	}

	var (
		reports    = make([]string, 0, len(job.partitioners))
		errs       = make([]string, 0, len(job.partitioners))
		itemsCount = make(map[string]int, job.cfg.NodesCount())
		wg         sync.WaitGroup
	)

	startTime := time.Now()

	for _, prt := range job.partitioners {
		wg.Add(1)

		go func(p *partitioner.Partitioner) {
			if err := p.Run(ctx); err != nil {
				errs = append(errs, fmt.Sprintf("[!] %s", err.Error()))
			}

			wg.Done()
		}(prt)
	}

	wg.Wait()

	finishTime := time.Since(startTime)

	fmt.Fprintf(os.Stderr, "Partitioning of %d yaml files finished in %d ms\n",
		len(job.partitioners), finishTime.Milliseconds())

	select {
	case <-ctx.Done():
		os.RemoveAll(job.cfg.WorkDir())

		return fmt.Errorf("context canceled: %w", ctx.Err())

	default:
		if err := filesutil.MoveDirAll(job.cfg.WorkDir(), *MainConfig.DstDirPath); err != nil {
			errs = append(errs, fmt.Sprintf("[!] %s", err.Error()))
		}
	}

	for _, p := range job.partitioners {
		reports = append(reports, fmt.Sprintf("===> %s", p.Report()))

		for shardName, count := range p.ShardItemsCount() {
			itemsCount[shardName] += count
		}
	}

	// for keeping sorted order of shards iterating over job.cfg.NodeNames()
	// instead of just iterating over itemsCount map.
	for i, name := range job.cfg.NodeNames() {
		// Skipping partitioning if ShardID has set
		if *MainConfig.ShardID >= 0 && *MainConfig.ShardID != i {
			continue
		}

		fmt.Fprintf(os.Stderr, "Shard %q got %d items in total\n", name, itemsCount[name])
	}

	if verbose && len(reports) > 0 {
		fmt.Fprintln(os.Stderr, strings.Join(reports, "\n"))
	}

	if len(errs) > 0 {
		return fmt.Errorf(
			"partitioning finished with %d error(s):\n%s",
			len(errs),
			strings.Join(errs, "\n"),
		)
	}

	return nil
}
