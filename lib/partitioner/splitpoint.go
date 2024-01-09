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
	"fmt"
	"strings"
)

func newSplitPoint(s string) (*splitPoint, error) {
	sp := strings.Split(s, ".")
	for i := 0; i < len(sp); i++ {
		elem := strings.TrimSpace(sp[i])

		if len(elem) == 0 {
			return nil, fmt.Errorf("invalid split point path: %q", s)
		}

		sp[i] = elem
	}

	return &splitPoint{slice: sp, str: strings.Join(sp, ".")}, nil
}

// splitPoint represents a path to a shardable yaml Node.
type splitPoint struct {
	str   string
	slice []string
}

// String implements a stringer interface.
func (sp *splitPoint) String() string { return sp.str }

// Slice returns the split point path as a slice of elements.
func (sp *splitPoint) Slice() []string { return sp.slice }

// Len returns the number of elements in the split point path.
func (sp *splitPoint) Len() int { return len(sp.slice) }

// Elem returns the split point path element with index i.
func (sp *splitPoint) Elem(i int) (string, error) {
	if i < 0 || i >= sp.Len() {
		return "", fmt.Errorf("index out of range: %d", i)
	}

	return sp.slice[i], nil
}
