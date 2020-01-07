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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLocalCache(t *testing.T, ttl time.Duration, eviction time.Duration) LocalCache {
	if ttl == 0 {
		ttl = time.Duration(time.Second)
	}
	if eviction == 0 {
		eviction = time.Duration(time.Second)
	}
	lcc := LocalCacheConfig{TTL: ttl, Eviction: eviction}
	cache, err := lcc.NewCache(&MockedCacheEncoder{}, nil)
	cache.Metrics = &MockCacheMetrics{}
	assert.Nil(t, err)
	return cache
}

func TestLocalInvalidShards(t *testing.T) {
	lcc := LocalCacheConfig{TTL: time.Duration(time.Second * 1), Shards: 3}
	_, err := lcc.NewCache(&MockedCacheEncoder{}, nil)
	assert.NotNil(t, err)
}

func TestLocalSetBytes(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	err := lc.SetBytes(context.Background(), "test-key", []byte("test-value"))
	assert.Nil(t, err)
}

func TestLocalSet(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("Set")
	value := "don't care"
	lc.Encoder.(*MockedCacheEncoder).On("Encode", value).Return([]byte("test-value"), nil)
	err := lc.Set(context.Background(), "test-key", value)
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "Set")
}

func TestLocalSetError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("SetCollision")
	value := "don't care"
	lc.Encoder.(*MockedCacheEncoder).On("Encode", value).Return(nil, fmt.Errorf("error"))
	err := lc.Set(context.Background(), "test-key", value)
	assert.Error(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "SetCollision")
}

func TestLocalGetBytes(t *testing.T) {
	lc := newLocalCache(t, 0, 0)

	// Use underlying cache to avoid testing two functions in one test
	err := lc.Cache.Set("test-key", []byte("test-value"))
	require.Nil(t, err)
	value, err := lc.GetBytes(context.Background(), "test-key")
	assert.Nil(t, err)
	assert.Equal(t, value, []byte("test-value"))
}

func TestLocalGet(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("Hit")

	// Use underlying cache to avoid testing two functions in one test
	err := lc.Cache.Set("test-key", []byte("test-value"))
	require.Nil(t, err)
	target := struct{}{}
	lc.Encoder.(*MockedCacheEncoder).On("Decode", []byte("test-value"), target).Return(nil)
	err = lc.Get(context.Background(), "test-key", target)
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "Hit")
}

func TestLocalGetError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("Miss")

	target := struct{}{}
	lc.Encoder.(*MockedCacheEncoder).On("Decode", []byte("test-value"), target).Return(fmt.Errorf("error"))
	err := lc.Get(context.Background(), "test-key", target)
	assert.Error(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "Miss")
}

func TestLocalGetBytesError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	value, err := lc.GetBytes(context.Background(), "test-key")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestLocalDelete(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("DeleteHit")

	// Use underlying cache to avoid testing two functions in one test
	err := lc.Cache.Set("test-key", []byte("test-value"))
	assert.Nil(t, err)
	err = lc.Delete(context.Background(), "test-key")
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "DeleteHit")
}

func TestLocalDeleteError(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("DeleteMiss")
	err := lc.Delete(context.Background(), "test-key")
	assert.Error(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "DeleteMiss")
}

func TestLocalPurge(t *testing.T) {
	lc := newLocalCache(t, 0, 0)
	lc.Metrics.(*MockCacheMetrics).On("PurgeHit")
	err := lc.Purge(context.Background())
	assert.Nil(t, err)
	lc.Metrics.(*MockCacheMetrics).AssertCalled(t, "PurgeHit")
}
