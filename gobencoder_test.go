// Copyright 2020 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tieredcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEncodable struct {
	A int
	B map[int]int
	C *testNestedEncodable
}

type testNestedEncodable struct {
	D string
}

// Test encoding and then decoding a search result from gob
func TestGobCacheEncoder_EncodeDecode(t *testing.T) {
	result := testEncodable{
		A: 5,
		B: map[int]int{0: 5, 1: 6, 2: 7},
		C: &testNestedEncodable{"thank"},
	}
	enc := GobCacheEncoder{}
	bytes, encodeErr := enc.Encode(result)
	require.Nil(t, encodeErr)
	decodedResult := testEncodable{}
	decodeErr := enc.Decode(bytes, &decodedResult)
	require.Nil(t, decodeErr)
	assert.Equal(t, result, decodedResult)
}

// Test that an error is returned if an error occurs encoding a search result
func TestGobCacheEncoder_Error(t *testing.T) {
	// nil pointer should cause gob to error
	enc := GobCacheEncoder{}
	_, err := enc.Encode(nil)
	assert.Error(t, err)
}
