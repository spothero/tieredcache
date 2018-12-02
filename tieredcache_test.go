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
	"context"
	"testing"

	"github.com/mna/redisc"
	"github.com/stretchr/testify/assert"
)

func TestTieredSetBytes(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	err := mtc.SetBytes(context.Background(), "test-key", []byte("test-value"))
	assert.Nil(t, err)
	localValue, ok := mtc.Local.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(localValue))
	remoteValue, ok := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(remoteValue))
}

func TestTieredSet(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	value := "don't care"
	encoder.On("Encode", value).Return([]byte("test-value"), nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Set")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	err := mtc.Set(context.Background(), "test-key", value)
	assert.Nil(t, err)
	localValue, ok := mtc.Local.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(localValue))
	remoteValue, ok := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.True(t, ok)
	assert.Equal(t, "test-value", string(remoteValue))
	mcm.AssertCalled(t, "Set")
}

func TestTieredGetBytesLocal(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	value, err := mtc.GetBytes(context.Background(), "test-key")
	assert.Nil(t, err)
	assert.Equal(t, "test-value", string(value))
}

func TestTieredGetLocal(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Hit")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Get(context.Background(), "test-key", target)
	assert.Nil(t, err)
	mcm.AssertCalled(t, "Hit")
}

func TestTieredGetBytesRemote(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	value, err := mtc.GetBytes(context.Background(), "test-key")
	assert.Nil(t, err)
	assert.Equal(t, "test-value", string(value))
}

func TestTieredGetRemote(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Hit")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Get(context.Background(), "test-key", target)
	assert.Nil(t, err)
	mcm.AssertCalled(t, "Hit")
}

// TestTieredGetError tests a fall-through on both local and remote
func TestTieredGetError(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Miss")
	mtc := TieredCache{
		Local:   NewMockCache(encoder),
		Remote:  NewMockCache(encoder),
		Metrics: mcm,
	}
	err := mtc.Get(context.Background(), "test-key", target)
	assert.Error(t, err)
	mcm.AssertCalled(t, "Miss")
}

func TestTieredGetBytesError(t *testing.T) {
	mtc := TieredCache{
		Local:  NewMockCache(nil),
		Remote: NewMockCache(nil),
	}
	value, err := mtc.GetBytes(context.Background(), "test-key")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestTieredDelete(t *testing.T) {
	mcm := &MockCacheMetrics{}
	mcm.On("DeleteHit")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  NewMockCache(nil),
		Metrics: mcm,
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Delete(context.Background(), "test-key")
	assert.Nil(t, err)
	localValue, localOk := mtc.Local.(*MockCache).Cache["test-key"]
	assert.False(t, localOk)
	assert.Nil(t, localValue)
	remoteValue, remoteOk := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.False(t, remoteOk)
	assert.Nil(t, remoteValue)
	mcm.AssertCalled(t, "DeleteHit")
}

func TestTieredDeleteError(t *testing.T) {
	mcm := &MockCacheMetrics{}
	mcm.On("DeleteMiss")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  NewMockCache(nil),
		Metrics: mcm,
	}
	err := mtc.Delete(context.Background(), "test-key")
	assert.NotNil(t, err)
	mcm.AssertCalled(t, "DeleteMiss")
}

func TestTieredPurge(t *testing.T) {
	mcm := &MockCacheMetrics{}
	mcm.On("PurgeHit")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  NewMockCache(nil),
		Metrics: mcm,
	}
	mtc.Local.(*MockCache).Cache["test-key"] = []byte("test-value")
	mtc.Remote.(*MockCache).Cache["test-key"] = []byte("test-value")
	err := mtc.Purge(context.Background())
	assert.Nil(t, err)
	localValue, localOk := mtc.Local.(*MockCache).Cache["test-key"]
	assert.False(t, localOk)
	assert.Nil(t, localValue)
	remoteValue, remoteOk := mtc.Remote.(*MockCache).Cache["test-key"]
	assert.False(t, remoteOk)
	assert.Nil(t, remoteValue)
	mcm.AssertCalled(t, "PurgeHit")
}

func TestTieredPurgeError(t *testing.T) {
	// Here we are just lazy -- the only way to get DEL to fail is to break the connection.
	// Given that, do not create a miniredis to connect to.
	mockCluster := &redisc.Cluster{StartupNodes: []string{""}}

	mcm := &MockCacheMetrics{}
	mcm.On("PurgeMiss")
	mtc := TieredCache{
		Local:   NewMockCache(nil),
		Remote:  RemoteCache{cluster: mockCluster},
		Metrics: mcm,
	}
	err := mtc.Purge(context.Background())
	assert.Error(t, err)
	mcm.AssertCalled(t, "PurgeMiss")
}
