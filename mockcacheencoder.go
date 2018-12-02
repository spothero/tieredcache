// Copyright 2018 SpotHero
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
	"github.com/stretchr/testify/mock"
)

// MockCacheEncoder mocks the cache encoder for use in tests
type MockedCacheEncoder struct {
	mock.Mock
}

// Encode mocks the cache encode implementation
func (mce *MockedCacheEncoder) Encode(value interface{}) ([]byte, error) {
	args := mce.Called(value)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Decode mocks the cache decode implementation
func (mce *MockedCacheEncoder) Decode(cachedValue []byte, target interface{}) error {
	args := mce.Called(cachedValue, target)
	return args.Error(0)
}
