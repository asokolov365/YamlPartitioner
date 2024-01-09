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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_splitPoint(t *testing.T) {
	f := func(s string) {
		t.Helper()

		sp, err := newSplitPoint(s)
		require.NoError(t, err)
		require.Equal(t, s, sp.String())
		require.Equal(t, strings.Split(s, "."), sp.Slice())

		_, err = sp.Elem(sp.Len() + 1)
		require.Error(t, err)

		_, err = sp.Elem(-1)
		require.Error(t, err)
	}

	f("groups.*.rules")
	f("module")
}

func Test_splitPointError(t *testing.T) {
	f := func(s string) {
		t.Helper()

		_, err := newSplitPoint(s)
		require.ErrorContains(t, err, "invalid split point path")
	}

	f("")
	f(". . . . . . . .")
	f("test..test")
}
