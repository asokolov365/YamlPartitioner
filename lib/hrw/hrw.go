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

package hrw

import (
	"fmt"
	"sync"

	"github.com/asokolov365/YamlPartitioner/lib/bytesutil"
)

// Rendezvous ...
// The original rendevouz hasing code is a taken from the following
// https://github.com/dgryski/go-rendezvous/blob/master/rdv.go (MIT License)
// The implementation optimizes the multiple hashing by pre-hashing
// the nodes and using an xorshift random number generator as a cheap integer hash function.
// A few bugs have been fixed and the hashing algorithm was changed to xxhash v2
type Rendezvous struct {
	hasher     Hasher
	nodes      map[string]int
	nodeHashes []uint64
	nodeNames  []string
	mu         sync.Mutex
}

// Hasher is a hash function suitable for general hash-based lookups.
// Example: xxhash.Sum64
//
//	func xxhash.Sum64(b []byte) uint64
//	Sum64 computes the 64-bit xxHash digest of input.
type Hasher func(input []byte) uint64

// New creates a new Rendezvous that implements Rendezvous
// or highest random weight (HRW) hashing algorithm
func New(hasher Hasher, nodes ...string) (*Rendezvous, error) {
	memo := make(map[string]struct{}, len(nodes))
	uniqNodes := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := memo[node]; ok {
			return nil, fmt.Errorf("duplicated node name: %s", node)
		}
		memo[node] = struct{}{}
		uniqNodes = append(uniqNodes, node)
	}

	r := &Rendezvous{
		hasher:     hasher,
		nodes:      make(map[string]int, len(uniqNodes)),
		nodeHashes: make([]uint64, len(uniqNodes)),
		nodeNames:  make([]string, len(uniqNodes)),
	}

	for i, node := range uniqNodes {
		r.nodes[node] = i
		r.nodeHashes[i] = hasher(bytesutil.ToUnsafeBytes(node))
		r.nodeNames[i] = node
	}

	return r, nil
}

// NodeNames returns the list of node names in the Rendezvous
func (r *Rendezvous) NodeNames() []string { return r.nodeNames }

// NodesCount returns the number of nodes in the Rendezvous
func (r *Rendezvous) NodesCount() int { return len(r.nodeNames) }

// Add adds nodes to the rendezvous
func (r *Rendezvous) Add(nodes ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range nodes {
		r.addNode(n)
	}
}

func (r *Rendezvous) addNode(node string) {
	if _, ok := r.nodes[node]; ok {
		return
	}
	r.nodes[node] = len(r.nodeNames) // set node idx
	r.nodeNames = append(r.nodeNames, node)
	r.nodeHashes = append(r.nodeHashes, r.hasher(bytesutil.ToUnsafeBytes(node)))
}

// Get gets the most suitable node name for a key
//
// Use bytesutil.ToUnsafeBytes(str) for fast
// string => []byte conversion
func (r *Rendezvous) Get(key []byte) string {
	if len(r.nodes) == 0 {
		return ""
	}
	nodeIdx := r.getNBestNodesForKey(key, 1)[0]
	return r.nodeNames[nodeIdx]
}

// GetN gets N most suitable node names for a key
//
// Use bytesutil.ToUnsafeBytes(str) for fast
// string => []byte conversion
func (r *Rendezvous) GetN(key []byte, replicasCount int) map[string]struct{} {
	if len(r.nodes) == 0 {
		return map[string]struct{}{}
	}
	nodeIndecies := r.getNBestNodesForKey(key, replicasCount)
	res := make(map[string]struct{}, len(nodeIndecies))
	for _, idx := range nodeIndecies {
		res[r.nodeNames[idx]] = struct{}{}
	}
	return res
}

func (r *Rendezvous) getNBestNodesForKey(key []byte, replicasCount int) []int {
	var nodeIndecies []int

	if len(r.nodes) == 1 {
		return []int{0}
	}

	if replicasCount >= len(r.nodes) {
		nodeIndecies = make([]int, len(r.nodes))
		for i := 0; i < len(r.nodes); i++ {
			nodeIndecies[i] = i
		}
		return nodeIndecies
	}
	if replicasCount < 1 {
		replicasCount = 1
	}

	keyHash := r.hasher(key)

	var maxIdx int
	var maxHash = xorshiftMult64(keyHash ^ r.nodeHashes[0]) // first node

	for i, nodeHash := range r.nodeHashes[1:] {
		if h := xorshiftMult64(keyHash ^ nodeHash); h > maxHash {
			maxIdx = i + 1
			maxHash = h
		}
	}

	nodeIndecies = make([]int, replicasCount)
	for i := 0; i < replicasCount; i++ {
		nodeIndecies[i] = maxIdx
		maxIdx++
		if maxIdx >= len(r.nodes) {
			maxIdx = 0
		}
	}
	return nodeIndecies
}

// Remove removes node from the rendezvous
func (r *Rendezvous) Remove(node string) {
	if len(r.nodes) == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// find index of node to remove
	nodeIdx := r.nodes[node]

	// remove from the slices
	lastIdx := len(r.nodeNames) - 1
	r.nodeNames[nodeIdx] = r.nodeNames[lastIdx]
	r.nodeNames = r.nodeNames[:lastIdx]

	r.nodeHashes[nodeIdx] = r.nodeHashes[lastIdx]
	r.nodeHashes = r.nodeHashes[:lastIdx]

	// update the map
	delete(r.nodes, node)
	moved := r.nodeNames[nodeIdx]
	r.nodes[moved] = nodeIdx
}

func xorshiftMult64(x uint64) uint64 {
	x ^= x >> 12 // a
	x ^= x << 25 // b
	x ^= x >> 27 // c
	return x * 2685821657736338717
}
