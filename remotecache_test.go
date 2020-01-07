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

	"github.com/alicebob/miniredis"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/assert"
)

func TestRemoteSetBytes(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster}
	err = rc.SetBytes(context.Background(), "test-key", []byte("test-value"))
	assert.NoError(t, err)
}

func TestRemoteSet(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	encoder := &MockedCacheEncoder{}
	value := "don't care"
	encoder.On("Encode", value).Return([]byte("test-value"), nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Set")
	rc := RemoteCache{cluster: mockCluster, Encoder: encoder, Metrics: mcm}
	err = rc.Set(context.Background(), "test-key", value)
	assert.NoError(t, err)
	mcm.AssertCalled(t, "Set")
}

func TestRemoteSetError(t *testing.T) {
	encoder := &MockedCacheEncoder{}
	value := "don't care"
	encoder.On("Encode", value).Return(nil, fmt.Errorf("error"))
	mcm := &MockCacheMetrics{}
	mcm.On("SetCollision")
	rc := RemoteCache{cluster: &redisc.Cluster{}, Encoder: encoder, Metrics: mcm}
	err := rc.Set(context.Background(), "test-key", value)
	assert.Error(t, err)
	mcm.AssertCalled(t, "SetCollision")
}

func TestRemoteGetBytes(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster}
	value, err := rc.GetBytes(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.Equal(t, value, []byte("test-value"))
}

func TestRemoteGet(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	encoder := &MockedCacheEncoder{}
	target := struct{}{}
	encoder.On("Decode", []byte("test-value"), target).Return(nil)
	mcm := &MockCacheMetrics{}
	mcm.On("Hit")
	rc := RemoteCache{cluster: mockCluster, Encoder: encoder, Metrics: mcm}
	err = rc.Get(context.Background(), "test-key", target)
	assert.NoError(t, err)
	mcm.AssertCalled(t, "Hit")
}

func TestRemoteGetError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	target := struct{}{}
	mcm := &MockCacheMetrics{}
	mcm.On("Miss")
	rc := RemoteCache{cluster: mockCluster, Metrics: mcm}
	err = rc.Get(context.Background(), "test-key", target)
	assert.Error(t, err)
	mcm.AssertCalled(t, "Miss")
}

func TestRemoteGetBytesError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("test-key-miss", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	rc := RemoteCache{cluster: mockCluster}
	value, err := rc.GetBytes(context.Background(), "test-key")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestRemoteDelete(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	s.Set("test-key", "test-value")
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	mcm := &MockCacheMetrics{}
	mcm.On("DeleteHit")
	rc := RemoteCache{cluster: mockCluster, Metrics: mcm}
	err = rc.Delete(context.Background(), "test-key")
	assert.NoError(t, err)
	mcm.AssertCalled(t, "DeleteHit")
}

func TestRemoteDeleteError(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	mcm := &MockCacheMetrics{}
	mcm.On("DeleteMiss")
	rc := RemoteCache{cluster: mockCluster, Metrics: mcm}
	err = rc.Delete(context.Background(), "test-key")
	assert.NoError(t, err)
	mcm.AssertCalled(t, "DeleteMiss")
}

func TestRemotePurge(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	mockCluster := &redisc.Cluster{StartupNodes: []string{s.Addr()}}

	mcm := &MockCacheMetrics{}
	mcm.On("PurgeHit")
	rc := RemoteCache{cluster: mockCluster, Metrics: mcm}
	err = rc.Purge(context.Background())
	assert.NoError(t, err)
	mcm.AssertCalled(t, "PurgeHit")
}

func TestRemotePurgeError(t *testing.T) {
	// Here we are just lazy -- the only way to get DEL to fail is to break the connection.
	// Given that, do not create a miniredis to connect to.
	mockCluster := &redisc.Cluster{StartupNodes: []string{""}}

	mcm := &MockCacheMetrics{}
	mcm.On("PurgeMiss")
	rc := RemoteCache{cluster: mockCluster, Metrics: mcm}
	err := rc.Purge(context.Background())
	assert.Error(t, err)
	mcm.AssertCalled(t, "PurgeMiss")
}
