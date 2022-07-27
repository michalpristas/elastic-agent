// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:dupl // duplicate code is in test cases
package capabilities

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent/internal/pkg/fleetapi"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

func TestUpgrade(t *testing.T) {
	l, _ := logger.New("test", false)
	t.Run("invalid rule", func(t *testing.T) {
		r := &inputCapability{}
		cap, err := newUpgradeCapability(l, r)
		assert.NoError(t, err, "no error expected")
		assert.Nil(t, cap, "cap should not be created")
	})

	t.Run("empty eql", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "",
				},
			},
		}

		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")
	})

	t.Run("valid action - version match", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "${version} == '8.0.0'",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "8.0.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.NoError(t, err, "should not be failing")
		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid action - deny version match", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "deny",
					UpgradeEqlDefinition: "${version} == '8.0.0'",
				},
			},
		}

		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "8.0.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.Error(t, err, "should fail")
		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid action - deny version match", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "deny",
					UpgradeEqlDefinition: "${version} == '8.*.*'",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "8.0.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.NoError(t, err, "should not fail")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid action - version mismmatch", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "${version} == '7.12.0'",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "8.0.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Error(t, err, "should fail")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid action - version bug allowed minor mismatch", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "match(${version}, '8.0.*')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "8.1.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Error(t, err, "should fail")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid action - version minor allowed major mismatch", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "match(${version}, '8.*.*')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "7.157.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Error(t, err, "should fail")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid action - version minor allowed minor upgrade", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "match(${version}, '8.*.*')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		ta := map[string]interface{}{
			"version": "8.2.0",
		}
		outAfter, err := cap.Apply(ta)

		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.NoError(t, err, "should not fail")
		assert.Equal(t, ta, outAfter)
	})

	t.Run("valid fleetatpi.action - version match", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "match(${version}, '8.*.*')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := fleetapi.ActionUpgrade{
			ActionID:   "",
			ActionType: "",
			Version:    "8.2.0",
			SourceURI:  "http://artifacts.elastic.co",
		}
		outAfter, err := cap.Apply(apiAction)

		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.NoError(t, err, "should not fail")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})

	t.Run("valid fleetatpi.action - version mismmatch", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "match(${version}, '8.*.*')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := map[string]interface{}{
			"version":    "9.0.0",
			"source_uri": "http://artifacts.elastic.co",
		}
		outAfter, err := cap.Apply(apiAction)

		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Error(t, err, "should fail")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})

	t.Run("valid fleetatpi.action - version mismmatch", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "match(${version}, '8.*.*')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := map[string]interface{}{
			"version":    "9.0.0",
			"source_uri": "http://artifacts.elastic.co",
		}
		outAfter, err := cap.Apply(apiAction)

		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Error(t, err, "should fail")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})

	t.Run("valid action - source uri trusted", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "startsWith(${source_uri}, 'https')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := map[string]interface{}{
			"version":    "9.0.0",
			"source_uri": "https://artifacts.elastic.co",
		}
		outAfter, err := cap.Apply(apiAction)

		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.NoError(t, err, "should not fail")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})

	t.Run("valid action - source uri untrusted", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "startsWith(${source_uri}, 'https')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := map[string]interface{}{
			"version":    "9.0.0",
			"source_uri": "http://artifacts.elastic.co",
		}
		outAfter, err := cap.Apply(apiAction)

		assert.Equal(t, ErrBlocked, err, "should be blocking")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})

	t.Run("unknown action", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&upgradeCapability{
					Type:                 "allow",
					UpgradeEqlDefinition: "startsWith(${source_uri}, 'https')",
				},
			},
		}
		cap, err := newUpgradesCapability(l, rd)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := fleetapi.ActionPolicyChange{}
		outAfter, err := cap.Apply(apiAction)

		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.NoError(t, err, "should not fail")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})
}
