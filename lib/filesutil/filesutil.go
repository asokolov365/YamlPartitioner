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

// Package filesutil implements utility routines for working with files.
package filesutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/bmatcuk/doublestar/v4"
)

// MoveDirAll reads srcDir entries and copies them into dstDir,
// then removes srcDir if no errors while copying.
func MoveDirAll(srcDir, dstDir string) error {
	if err := CopyDirAll(srcDir, dstDir); err != nil {
		return err
	}

	if err := os.RemoveAll(srcDir); err != nil {
		return fmt.Errorf("failed to remove %q: %w", srcDir, err)
	}

	return nil
}

// CopyDirAll copies the entire subtree connected at srcDir into dstDir.
// Note: the contents of the srcDir are copied rather than the directory itself.
// This also causes symbolic links to be copied, rather than indirected through.
func CopyDirAll(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %q: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		fileInfo, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("failed to get file info %q: %w", srcPath, err)
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for %q", srcPath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := createIfNotExists(dstPath, 0o755); err != nil {
				return err
			}

			if err := CopyDirAll(srcPath, dstPath); err != nil {
				return err
			}

		case os.ModeSymlink:
			if err := CopySymLink(srcPath, dstPath); err != nil {
				return err
			}

		default:
			if err := CopyContent(srcPath, dstPath); err != nil {
				return err
			}
		}

		if err := os.Lchown(dstPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return fmt.Errorf("failed to change owner %q: %w", dstPath, err)
		}

		srcInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info %q: %w", entry.Name(), err)
		}

		isSymlink := srcInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(dstPath, srcInfo.Mode()); err != nil {
				return fmt.Errorf("failed to change mode %q: %w", dstPath, err)
			}
		}
	}

	return nil
}

// CopyContent copies srcFile to dstFile with io.Copy.
func CopyContent(srcFile, dstFile string) error {
	out, err := os.OpenFile(dstFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create %q: %w", dstFile, err)
	}
	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", srcFile, err)
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("failed to copy %q to %q: %w", in.Name(), out.Name(), err)
	}

	return nil
}

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func createIfNotExists(dir string, perm os.FileMode) error {
	if exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to make directory %q: %w", dir, err)
	}

	return nil
}

// CopySymLink reads a target of the src symlink and
// creates a dst symlink pointing to the target.
func CopySymLink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("failed to read symlink %q: %w", src, err)
	}

	if err := os.Symlink(target, dst); err != nil {
		return fmt.Errorf("failed to create symlink %q: %w", dst, err)
	}

	return nil
}

// LongestCommonPath returns the longest common path for
// a given list of file paths.
func LongestCommonPath(paths []string) string {
	commonPrefix := longestCommonPrefix(paths)
	idx := strings.LastIndex(commonPrefix, "/")

	return commonPrefix[:idx+1]
}

func longestCommonPrefix(strs []string) string {
	switch {
	case len(strs) == 0:
		return ""
	case len(strs) == 1:
		return strs[0]
	}

	sort.Strings(strs)

	first := strs[0]
	last := strs[len(strs)-1]

	maxLen := len(first)
	if len(last) < maxLen {
		maxLen = len(last)
	}

	var longestPrefix strings.Builder

	stop := false

	for i := 0; i < maxLen && !stop; i++ {
		if first[i] == last[i] {
			longestPrefix.WriteByte(first[i])
		} else {
			stop = true
		}
	}

	return longestPrefix.String()
}

// List returns the absolute representation of all
// files matching pattern or nil if there is no matching file.
func List(pattern string) ([]string, error) {
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to match files with pattern %s: %w", pattern, err)
	}

	for i := 0; i < len(matches); i++ {
		absPath, err := filepath.Abs(matches[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %q: %w", matches[i], err)
		}

		matches[i] = absPath
	}

	return matches, nil
}
