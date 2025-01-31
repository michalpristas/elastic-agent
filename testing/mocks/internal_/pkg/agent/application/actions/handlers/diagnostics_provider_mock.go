// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

// Code generated by mockery v2.51.1. DO NOT EDIT.

package handlers

import (
	context "context"

	component "github.com/elastic/elastic-agent/pkg/component"

	cproto "github.com/elastic/elastic-agent/pkg/control/v2/cproto"

	diagnostics "github.com/elastic/elastic-agent/internal/pkg/diagnostics"

	mock "github.com/stretchr/testify/mock"

	runtime "github.com/elastic/elastic-agent/pkg/component/runtime"
)

// DiagnosticsProvider is an autogenerated mock type for the diagnosticsProvider type
type DiagnosticsProvider struct {
	mock.Mock
}

type DiagnosticsProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *DiagnosticsProvider) EXPECT() *DiagnosticsProvider_Expecter {
	return &DiagnosticsProvider_Expecter{mock: &_m.Mock}
}

// DiagnosticHooks provides a mock function with no fields
func (_m *DiagnosticsProvider) DiagnosticHooks() diagnostics.Hooks {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for DiagnosticHooks")
	}

	var r0 diagnostics.Hooks
	if rf, ok := ret.Get(0).(func() diagnostics.Hooks); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(diagnostics.Hooks)
		}
	}

	return r0
}

// DiagnosticsProvider_DiagnosticHooks_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DiagnosticHooks'
type DiagnosticsProvider_DiagnosticHooks_Call struct {
	*mock.Call
}

// DiagnosticHooks is a helper method to define mock.On call
func (_e *DiagnosticsProvider_Expecter) DiagnosticHooks() *DiagnosticsProvider_DiagnosticHooks_Call {
	return &DiagnosticsProvider_DiagnosticHooks_Call{Call: _e.mock.On("DiagnosticHooks")}
}

func (_c *DiagnosticsProvider_DiagnosticHooks_Call) Run(run func()) *DiagnosticsProvider_DiagnosticHooks_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DiagnosticsProvider_DiagnosticHooks_Call) Return(_a0 diagnostics.Hooks) *DiagnosticsProvider_DiagnosticHooks_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DiagnosticsProvider_DiagnosticHooks_Call) RunAndReturn(run func() diagnostics.Hooks) *DiagnosticsProvider_DiagnosticHooks_Call {
	_c.Call.Return(run)
	return _c
}

// PerformComponentDiagnostics provides a mock function with given fields: ctx, additionalMetrics, req
func (_m *DiagnosticsProvider) PerformComponentDiagnostics(ctx context.Context, additionalMetrics []cproto.AdditionalDiagnosticRequest, req ...component.Component) ([]runtime.ComponentDiagnostic, error) {
	_va := make([]interface{}, len(req))
	for _i := range req {
		_va[_i] = req[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, additionalMetrics)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for PerformComponentDiagnostics")
	}

	var r0 []runtime.ComponentDiagnostic
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []cproto.AdditionalDiagnosticRequest, ...component.Component) ([]runtime.ComponentDiagnostic, error)); ok {
		return rf(ctx, additionalMetrics, req...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []cproto.AdditionalDiagnosticRequest, ...component.Component) []runtime.ComponentDiagnostic); ok {
		r0 = rf(ctx, additionalMetrics, req...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]runtime.ComponentDiagnostic)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []cproto.AdditionalDiagnosticRequest, ...component.Component) error); ok {
		r1 = rf(ctx, additionalMetrics, req...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DiagnosticsProvider_PerformComponentDiagnostics_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PerformComponentDiagnostics'
type DiagnosticsProvider_PerformComponentDiagnostics_Call struct {
	*mock.Call
}

// PerformComponentDiagnostics is a helper method to define mock.On call
//   - ctx context.Context
//   - additionalMetrics []cproto.AdditionalDiagnosticRequest
//   - req ...component.Component
func (_e *DiagnosticsProvider_Expecter) PerformComponentDiagnostics(ctx interface{}, additionalMetrics interface{}, req ...interface{}) *DiagnosticsProvider_PerformComponentDiagnostics_Call {
	return &DiagnosticsProvider_PerformComponentDiagnostics_Call{Call: _e.mock.On("PerformComponentDiagnostics",
		append([]interface{}{ctx, additionalMetrics}, req...)...)}
}

func (_c *DiagnosticsProvider_PerformComponentDiagnostics_Call) Run(run func(ctx context.Context, additionalMetrics []cproto.AdditionalDiagnosticRequest, req ...component.Component)) *DiagnosticsProvider_PerformComponentDiagnostics_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]component.Component, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(component.Component)
			}
		}
		run(args[0].(context.Context), args[1].([]cproto.AdditionalDiagnosticRequest), variadicArgs...)
	})
	return _c
}

func (_c *DiagnosticsProvider_PerformComponentDiagnostics_Call) Return(_a0 []runtime.ComponentDiagnostic, _a1 error) *DiagnosticsProvider_PerformComponentDiagnostics_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DiagnosticsProvider_PerformComponentDiagnostics_Call) RunAndReturn(run func(context.Context, []cproto.AdditionalDiagnosticRequest, ...component.Component) ([]runtime.ComponentDiagnostic, error)) *DiagnosticsProvider_PerformComponentDiagnostics_Call {
	_c.Call.Return(run)
	return _c
}

// PerformDiagnostics provides a mock function with given fields: ctx, req
func (_m *DiagnosticsProvider) PerformDiagnostics(ctx context.Context, req ...runtime.ComponentUnitDiagnosticRequest) []runtime.ComponentUnitDiagnostic {
	_va := make([]interface{}, len(req))
	for _i := range req {
		_va[_i] = req[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for PerformDiagnostics")
	}

	var r0 []runtime.ComponentUnitDiagnostic
	if rf, ok := ret.Get(0).(func(context.Context, ...runtime.ComponentUnitDiagnosticRequest) []runtime.ComponentUnitDiagnostic); ok {
		r0 = rf(ctx, req...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]runtime.ComponentUnitDiagnostic)
		}
	}

	return r0
}

// DiagnosticsProvider_PerformDiagnostics_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PerformDiagnostics'
type DiagnosticsProvider_PerformDiagnostics_Call struct {
	*mock.Call
}

// PerformDiagnostics is a helper method to define mock.On call
//   - ctx context.Context
//   - req ...runtime.ComponentUnitDiagnosticRequest
func (_e *DiagnosticsProvider_Expecter) PerformDiagnostics(ctx interface{}, req ...interface{}) *DiagnosticsProvider_PerformDiagnostics_Call {
	return &DiagnosticsProvider_PerformDiagnostics_Call{Call: _e.mock.On("PerformDiagnostics",
		append([]interface{}{ctx}, req...)...)}
}

func (_c *DiagnosticsProvider_PerformDiagnostics_Call) Run(run func(ctx context.Context, req ...runtime.ComponentUnitDiagnosticRequest)) *DiagnosticsProvider_PerformDiagnostics_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]runtime.ComponentUnitDiagnosticRequest, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(runtime.ComponentUnitDiagnosticRequest)
			}
		}
		run(args[0].(context.Context), variadicArgs...)
	})
	return _c
}

func (_c *DiagnosticsProvider_PerformDiagnostics_Call) Return(_a0 []runtime.ComponentUnitDiagnostic) *DiagnosticsProvider_PerformDiagnostics_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DiagnosticsProvider_PerformDiagnostics_Call) RunAndReturn(run func(context.Context, ...runtime.ComponentUnitDiagnosticRequest) []runtime.ComponentUnitDiagnostic) *DiagnosticsProvider_PerformDiagnostics_Call {
	_c.Call.Return(run)
	return _c
}

// NewDiagnosticsProvider creates a new instance of DiagnosticsProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDiagnosticsProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *DiagnosticsProvider {
	mock := &DiagnosticsProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
