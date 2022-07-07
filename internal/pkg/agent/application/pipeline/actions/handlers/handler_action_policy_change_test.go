// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package handlers

import (
	"context"
	"sync"
	"testing"

	"github.com/elastic/elastic-agent/internal/pkg/agent/application/info"
	"github.com/elastic/elastic-agent/internal/pkg/agent/configuration"
	"github.com/elastic/elastic-agent/internal/pkg/agent/storage"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/internal/pkg/config"
	"github.com/elastic/elastic-agent/internal/pkg/fleetapi"
	noopacker "github.com/elastic/elastic-agent/internal/pkg/fleetapi/acker/noop"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

type mockEmitter struct {
	err    error
	policy *config.Config
}

func (m *mockEmitter) Emitter(_ context.Context, policy *config.Config) error {
	m.policy = policy
	return m.err
}

func TestPolicyChange(t *testing.T) {
	log, _ := logger.New("", false)
	ack := noopacker.NewAcker()
	agentInfo, _ := info.NewAgentInfo(true)
	nullStore := &storage.NullStore{}

	t.Run("Receive a config change and successfully emits a raw configuration", func(t *testing.T) {
		emitter := &mockEmitter{}

		conf := map[string]interface{}{"hello": "world"}
		action := &fleetapi.ActionPolicyChange{
			ActionID:   "abc123",
			ActionType: "POLICY_CHANGE",
			Policy:     conf,
		}

		cfg := configuration.DefaultConfiguration()
		handler := &PolicyChange{
			log:       log,
			emitter:   emitter.Emitter,
			agentInfo: agentInfo,
			config:    cfg,
			store:     nullStore,
		}

		err := handler.Handle(context.Background(), action, ack)
		require.NoError(t, err)
		require.Equal(t, config.MustNewConfigFrom(conf), emitter.policy)
	})

	t.Run("Receive a config and fail to emits a raw configuration", func(t *testing.T) {
		mockErr := errors.New("error returned")
		emitter := &mockEmitter{err: mockErr}

		conf := map[string]interface{}{"hello": "world"}
		action := &fleetapi.ActionPolicyChange{
			ActionID:   "abc123",
			ActionType: "POLICY_CHANGE",
			Policy:     conf,
		}

		cfg := configuration.DefaultConfiguration()
		handler := &PolicyChange{
			log:       log,
			emitter:   emitter.Emitter,
			agentInfo: agentInfo,
			config:    cfg,
			store:     nullStore,
		}

		err := handler.Handle(context.Background(), action, ack)
		require.Error(t, err)
	})

	t.Run("Receive and handle source_uri correctly", func(t *testing.T) {
		emitter := &mockEmitter{}

		conf := map[string]interface{}{
			"agent.download.sourceURI": "test",
		}
		action := &fleetapi.ActionPolicyChange{
			ActionID:   "abc123",
			ActionType: "POLICY_CHANGE",
			Policy:     conf,
		}

		cfg := configuration.DefaultConfiguration()
		handler := &PolicyChange{
			log:       log,
			emitter:   emitter.Emitter,
			agentInfo: agentInfo,
			config:    cfg,
			store:     nullStore,
			reloaders: map[string]ReloadFunc{
				"agent.download.sourceURI": func(value interface{}) error {
					if strVal, ok := value.(string); !ok {
						return errors.New("provided soruce_uri is not a string")
					} else if strVal != "" {
						cfg.Settings.DownloadConfig.SourceURI = strVal
					}
					return nil
				},
			},
		}

		err := handler.Handle(context.Background(), action, ack)
		require.NoError(t, err)
		require.Equal(t, "test", cfg.Settings.DownloadConfig.SourceURI)
	})

	t.Run("Receive and handle source_uri correctly, broken keys", func(t *testing.T) {
		emitter := &mockEmitter{}

		conf := map[string]interface{}{
			"agent": map[string]interface{}{
				"download": map[string]interface{}{
					"sourceURI": "test",
				},
			},
		}
		action := &fleetapi.ActionPolicyChange{
			ActionID:   "abc123",
			ActionType: "POLICY_CHANGE",
			Policy:     conf,
		}

		cfg := configuration.DefaultConfiguration()
		handler := &PolicyChange{
			log:       log,
			emitter:   emitter.Emitter,
			agentInfo: agentInfo,
			config:    cfg,
			store:     nullStore,
			reloaders: map[string]ReloadFunc{
				"agent.download.sourceURI": func(value interface{}) error {
					if strVal, ok := value.(string); !ok {
						return errors.New("provided soruce_uri is not a string")
					} else if strVal != "" {
						cfg.Settings.DownloadConfig.SourceURI = strVal
					}
					return nil
				},
			},
		}

		err := handler.Handle(context.Background(), action, ack)
		require.NoError(t, err)
		require.Equal(t, "test", cfg.Settings.DownloadConfig.SourceURI)
	})
}

func TestPolicyAcked(t *testing.T) {
	log, _ := logger.New("", false)
	agentInfo, _ := info.NewAgentInfo(true)
	nullStore := &storage.NullStore{}

	t.Run("Config change should not ACK on error", func(t *testing.T) {
		tacker := &testAcker{}

		mockErr := errors.New("error returned")
		emitter := &mockEmitter{err: mockErr}

		config := map[string]interface{}{"hello": "world"}
		actionID := "abc123"
		action := &fleetapi.ActionPolicyChange{
			ActionID:   actionID,
			ActionType: "POLICY_CHANGE",
			Policy:     config,
		}

		cfg := configuration.DefaultConfiguration()
		handler := &PolicyChange{
			log:       log,
			emitter:   emitter.Emitter,
			agentInfo: agentInfo,
			config:    cfg,
			store:     nullStore,
		}

		err := handler.Handle(context.Background(), action, tacker)
		require.Error(t, err)

		actions := tacker.Items()
		assert.EqualValues(t, 0, len(actions))
	})

	t.Run("Config change should ACK", func(t *testing.T) {
		tacker := &testAcker{}

		emitter := &mockEmitter{}

		config := map[string]interface{}{"hello": "world"}
		actionID := "abc123"
		action := &fleetapi.ActionPolicyChange{
			ActionID:   actionID,
			ActionType: "POLICY_CHANGE",
			Policy:     config,
		}

		cfg := configuration.DefaultConfiguration()
		handler := &PolicyChange{
			log:       log,
			emitter:   emitter.Emitter,
			agentInfo: agentInfo,
			config:    cfg,
			store:     nullStore,
		}

		err := handler.Handle(context.Background(), action, tacker)
		require.NoError(t, err)

		actions := tacker.Items()
		assert.EqualValues(t, 1, len(actions))
		assert.Equal(t, actionID, actions[0])
	})
}

type testAcker struct {
	acked     []string
	ackedLock sync.Mutex
}

func (t *testAcker) Ack(_ context.Context, action fleetapi.Action) error {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()

	if t.acked == nil {
		t.acked = make([]string, 0)
	}

	t.acked = append(t.acked, action.ID())
	return nil
}

func (t *testAcker) Commit(_ context.Context) error {
	return nil
}

func (t *testAcker) Clear() {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()

	t.acked = make([]string, 0)
}

func (t *testAcker) Items() []string {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()
	return t.acked
}
