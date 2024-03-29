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

package bytesutil

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToUnsafeString(t *testing.T) {
	t.Parallel()

	b := []byte("str")
	require.Equal(t, "str", ToUnsafeString(b))
}

func TestToUnsafeBytes(t *testing.T) {
	t.Parallel()

	s := "str"
	if !bytes.Equal([]byte("str"), ToUnsafeBytes(s)) {
		t.Fatalf(`[]bytes(%s) doesnt equal to %s `, s, s)
	}
}
