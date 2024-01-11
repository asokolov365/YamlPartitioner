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
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/asokolov365/YamlPartitioner/lib/bytesutil"
	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/require"
)

func getRandomInt(min, max int) int {
	// calculate the max we will be using
	bg := big.NewInt(int64(max - min))
	// get big.Int between 0 and bg
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		panic(err.Error())
	}
	// add n to min to support the passed in range
	return int(n.Int64() + int64(min))
}

func randStringAsBytes() []byte {
	length := getRandomInt(4, 64)
	return bytesutil.RandStringAsBytes(length)
}

func TestEmpty(t *testing.T) {
	t.Parallel()

	r, err := New(xxhash.Sum64)
	require.NoError(t, err)
	require.Empty(t, r.Get([]byte("hello")))
	require.Empty(t, r.GetN([]byte("hello"), 1))
	r.Remove("node1")
}

func TestNew_Okay(t *testing.T) {
	t.Parallel()

	nodeNum := 5
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)
	require.Equal(t, nodeNum, r.NodesCount())
	require.Equal(t, nodeNum, len(r.nodes))
	require.Equal(t, nodeNum, len(r.nodeNames))
	require.Equal(t, nodeNum, len(r.nodeHashes))
	require.Equal(t, nodes, r.NodeNames())
}

func TestNew_WithDuplicates(t *testing.T) {
	t.Parallel()

	nodes := []string{"node1", "node2", "node3", "node1", "node2", "node1"}
	_, err := New(xxhash.Sum64, nodes...)
	require.Error(t, err)
	require.ErrorContains(t, err, "duplicated node name:")
}

func TestAdd(t *testing.T) {
	t.Parallel()

	nodeNum := 5
	r, err := New(xxhash.Sum64)
	require.NoError(t, err)

	for i := 0; i < nodeNum; i++ {
		r.Add(fmt.Sprintf("node%d", i))
	}
	require.Equal(t, nodeNum, r.NodesCount())
	require.Equal(t, nodeNum, len(r.nodes))
	require.Equal(t, nodeNum, len(r.nodeNames))
	require.Equal(t, nodeNum, len(r.nodeHashes))

	// Add once again
	for i := 0; i < nodeNum; i++ {
		r.Add(fmt.Sprintf("node%d", i))
	}
	require.Equal(t, nodeNum, r.NodesCount())
	require.Equal(t, nodeNum, len(r.nodes))
	require.Equal(t, nodeNum, len(r.nodeNames))
	require.Equal(t, nodeNum, len(r.nodeHashes))
}

func TestRemove(t *testing.T) {
	t.Parallel()

	nodeNum := 5
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)
	require.Equal(t, nodeNum, len(r.nodes))
	require.Equal(t, nodeNum, len(r.nodeNames))
	require.Equal(t, nodeNum, len(r.nodeHashes))

	r.Remove("node2")
	// fmt.Printf("%+v\n", r.nodes)
	require.Equal(t, 2, r.nodes["node4"])
	require.Equal(t, "node4", r.nodeNames[2])
	require.Equal(t, nodeNum-1, len(r.nodes))
	require.Equal(t, nodeNum-1, len(r.nodeNames))
	require.Equal(t, nodeNum-1, len(r.nodeHashes))

	r.Remove("node1")
	// fmt.Printf("%+v\n", r.nodes)
	require.Equal(t, 1, r.nodes["node3"])
	require.Equal(t, "node3", r.nodeNames[1])
	require.Equal(t, nodeNum-2, len(r.nodes))
	require.Equal(t, nodeNum-2, len(r.nodeNames))
	require.Equal(t, nodeNum-2, len(r.nodeHashes))
}

func TestDistributeOver1(t *testing.T) {
	t.Parallel()

	nodeName := "default"
	nodes := []string{nodeName}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	numKeys := 100
	buckets := map[string]int{nodeName: 0}

	for i := 0; i < numKeys; i++ {
		n := r.Get(randStringAsBytes())
		require.NotEmpty(t, n)
		buckets[n]++
	}
	require.Equal(t, numKeys, buckets[nodeName])
}

func TestGetN(t *testing.T) {
	t.Parallel()

	nodeNum := 3
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	numKeys := 10
	buckets := make(map[string]int, nodeNum)

	var retNodes map[string]struct{}

	for i := 0; i < numKeys; i++ {
		retNodes = r.GetN(randStringAsBytes(), 5)
		require.Equal(t, nodeNum, len(retNodes))

		for node := range retNodes {
			buckets[node]++
		}
	}

	for _, node := range nodes {
		require.Equal(t, numKeys, buckets[node])
	}
}

func TestDistributeOver5(t *testing.T) {
	t.Parallel()

	nodeNum := 5
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	numKeys := 10000
	buckets := make(map[string]int, nodeNum)

	var node string

	for i := 0; i < numKeys; i++ {
		switch {
		case i%2 == 0:
			node = r.Get(randStringAsBytes())

		case i%3 == 0:
			retNodes := r.GetN(randStringAsBytes(), 0)
			require.Equal(t, 1, len(retNodes))

			for n := range retNodes {
				node = n
			}

		default:
			retNodes := r.GetN(randStringAsBytes(), 1)
			require.Equal(t, 1, len(retNodes))

			for n := range retNodes {
				node = n
			}
		}

		require.NotEmpty(t, node)
		buckets[node]++
	}

	lowerThreshold := int(float32(numKeys) * 0.15)
	higherThreshold := int(float32(numKeys) * 0.25)

	for n, l := range buckets {
		// fmt.Printf("%s got %d\n", n, l)
		require.Less(t, lowerThreshold, l,
			fmt.Sprintf("%q got less than 15%% of keys: %d < %d", n, l, lowerThreshold))
		require.Less(t, l, higherThreshold,
			fmt.Sprintf("%q got more than 25%% of keys: %d > %d", n, l, higherThreshold))
	}
}

func TestDistributeOver8(t *testing.T) {
	t.Parallel()

	nodeNum := 8
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	numKeys := 10000
	buckets := make(map[string]int, nodeNum)

	var node string

	for i := 0; i < numKeys; i++ {
		switch {
		case i%2 == 0:
			node = r.Get(randStringAsBytes())

		case i%3 == 0:
			retNodes := r.GetN(randStringAsBytes(), 0)
			require.Equal(t, 1, len(retNodes))

			for n := range retNodes {
				node = n
			}

		default:
			retNodes := r.GetN(randStringAsBytes(), 1)
			require.Equal(t, 1, len(retNodes))

			for n := range retNodes {
				node = n
			}
		}

		require.NotEmpty(t, node)
		buckets[node]++
	}

	lowerThreshold := int(float32(numKeys) * 0.11)
	higherThreshold := int(float32(numKeys) * 0.16)

	for n, l := range buckets {
		// fmt.Printf("%s got %d\n", n, l)
		require.Less(t, lowerThreshold, l,
			fmt.Sprintf("%q got less than 11%% of keys: %d < %d", n, l, lowerThreshold))
		require.Less(t, l, higherThreshold,
			fmt.Sprintf("%q got more than 16%% of keys: %d > %d", n, l, higherThreshold))
	}
}

func TestSameDistribution(t *testing.T) {
	t.Parallel()

	nodeNum := 8
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r1, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	r2, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	numKeys := 10000
	buckets1 := make(map[string]int, nodeNum)
	buckets2 := make(map[string]int, nodeNum)

	for i := 0; i < numKeys; i++ {
		key := randStringAsBytes()
		// fmt.Println(key)
		n1 := r1.Get(key)
		require.NotEmpty(t, n1)

		n2 := r2.Get(key)
		require.NotEmpty(t, n2)
		require.Equal(t, n1, n2)

		buckets1[n1]++
		buckets2[n2]++
	}

	require.Equal(t, len(buckets1), len(buckets2))

	for n := range buckets1 {
		require.Equal(t, buckets1[n], buckets2[n])
	}
}

func TestMovers(t *testing.T) {
	t.Parallel()

	nodeNum := 9
	nodes := make([]string, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodes[i] = fmt.Sprintf("node%d", i)
	}

	r, err := New(xxhash.Sum64, nodes...)
	require.NoError(t, err)

	numKeys := 10000
	kaysAlloc := make(map[string]string, numKeys)

	moversThreshold := int(float32(numKeys) / float32(nodeNum))

	for i := 0; i < numKeys; i++ {
		key := randStringAsBytes()
		n := r.Get(key)
		kaysAlloc[string(key)] = n
	}

	newNode := fmt.Sprintf("node%d", len(nodes))
	r.Add(newNode)
	// nodeNum++

	totalMovers := 0
	unnecessaryMovers := 0

	for key, prevNode := range kaysAlloc {
		n := r.Get(bytesutil.ToUnsafeBytes(key))
		if n != prevNode {
			totalMovers++

			if n != newNode {
				unnecessaryMovers++
			}
		}
	}

	require.Equal(t, 0, unnecessaryMovers)
	require.Less(t, totalMovers, moversThreshold)
}
