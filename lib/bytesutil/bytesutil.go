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

// Package bytesutil implements utility routines for manipulating byte slices.
package bytesutil

import (
	"unsafe"
)

// ToUnsafeString converts b to string without memory allocations.
//
// The returned string is valid only until b is reachable and unmodified.
func ToUnsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// ToUnsafeBytes converts s to a byte slice without memory allocations.
//
// The returned byte slice is valid only until s is reachable and unmodified.
func ToUnsafeBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// func ToUnsafeBytes(s string) (b []byte) {
// 	strh := (*reflect.StringHeader)(unsafe.Pointer(&s))
// 	slh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
// 	slh.Data = strh.Data
// 	slh.Len = strh.Len
// 	slh.Cap = strh.Len
// 	return b
// }
