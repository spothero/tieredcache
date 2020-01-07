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
	"bytes"
	"encoding/gob"
)

// CacheEncoder defines an interface for encoding and decoding
// values stored in cache
type CacheEncoder interface {
	// value must be a pointer
	Encode(value interface{}) ([]byte, error)
	// target must be a pointer
	Decode(cachedValue []byte, target interface{}) error
}

// GobCacheEncoder uses encoding/gob to encode values for caching
type GobCacheEncoder struct{}

// Encode encodes the provided value using gob. value must be a pointer.
func (gb *GobCacheEncoder) Encode(value interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes the cached value using the gob encoding and sets the
// result in target. target must be a pointer
func (gb *GobCacheEncoder) Decode(cachedValue []byte, target interface{}) error {
	reader := bytes.NewReader(cachedValue)
	dec := gob.NewDecoder(reader)
	return dec.Decode(target)
}
