// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/internal/pkg/agent/application/coordinator"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/gateway"
	"github.com/elastic/elastic-agent/internal/pkg/agent/storage"
	"github.com/elastic/elastic-agent/internal/pkg/agent/storage/store"
	"github.com/elastic/elastic-agent/internal/pkg/fleetapi"
	"github.com/elastic/elastic-agent/internal/pkg/fleetapi/acker"
	"github.com/elastic/elastic-agent/internal/pkg/fleetapi/acker/noop"
	"github.com/elastic/elastic-agent/internal/pkg/scheduler"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

type clientCallbackFunc func(headers http.Header, body io.Reader) (*http.Response, error)

type testingClient struct {
	sync.Mutex
	callback clientCallbackFunc
	received chan struct{}
}

func (t *testingClient) Send(
	_ context.Context,
	_ string,
	_ string,
	_ url.Values,
	headers http.Header,
	body io.Reader,
) (*http.Response, error) {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
	return t.callback(headers, body)
}

func (t *testingClient) URI() string {
	return "http://localhost"
}

func (t *testingClient) Answer(fn clientCallbackFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
	return t.received
}

func newTestingClient() *testingClient {
	return &testingClient{received: make(chan struct{}, 1)}
}

type testingDispatcherFunc func(...fleetapi.Action) error

type testingDispatcher struct {
	sync.Mutex
	callback testingDispatcherFunc
	received chan struct{}
}

func (t *testingDispatcher) Dispatch(_ context.Context, acker acker.Acker, actions ...fleetapi.Action) error {
	t.Lock()
	defer t.Unlock()
	defer func() { t.received <- struct{}{} }()
	// Get a dummy context.
	ctx := context.Background()

	// In context of testing we need to abort on error.
	if err := t.callback(actions...); err != nil {
		return err
	}

	// Ack everything and commit at the end.
	for _, action := range actions {
		_ = acker.Ack(ctx, action)
	}
	_ = acker.Commit(ctx)

	return nil
}

func (t *testingDispatcher) Answer(fn testingDispatcherFunc) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	t.callback = fn
	return t.received
}

func newTestingDispatcher() *testingDispatcher {
	return &testingDispatcher{received: make(chan struct{}, 1)}
}

type mockQueue struct {
	mock.Mock
}

func (m *mockQueue) Add(action fleetapi.Action, n int64) {
	m.Called(action, n)
}

func (m *mockQueue) DequeueActions() []fleetapi.Action {
	args := m.Called()
	return args.Get(0).([]fleetapi.Action)
}

func (m *mockQueue) Cancel(id string) int {
	args := m.Called(id)
	return args.Int(0)
}

func (m *mockQueue) Actions() []fleetapi.Action {
	args := m.Called()
	return args.Get(0).([]fleetapi.Action)
}

type withGatewayFunc func(*testing.T, gateway.FleetGateway, *testingClient, *testingDispatcher, *scheduler.Stepper)

func withGateway(agentInfo agentInfo, settings *fleetGatewaySettings, fn withGatewayFunc) func(t *testing.T) {
	return func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		log, _ := logger.New("fleet_gateway", false)

		stateStore := newStateStore(t, log)

		queue := &mockQueue{}
		queue.On("DequeueActions").Return([]fleetapi.Action{})
		queue.On("Actions").Return([]fleetapi.Action{})

		gateway, err := newFleetGatewayWithScheduler(
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)

		require.NoError(t, err)

		fn(t, gateway, client, dispatcher, scheduler)
	}
}

func ackSeq(channels ...<-chan struct{}) func() {
	return func() {
		for _, c := range channels {
			<-c
		}
	}
}

func wrapStrToResp(code int, body string) *http.Response {
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", code, http.StatusText(code)),
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}
}

func TestFleetGateway(t *testing.T) {
	agentInfo := &testAgentInfo{}
	settings := &fleetGatewaySettings{
		Duration: 5 * time.Second,
		Backoff:  backoffSettings{Init: 1 * time.Second, Max: 5 * time.Second},
	}

	t.Run("send no event and receive no action", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway gateway.FleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
	) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		// Synchronize scheduler and acking of calls from the worker go routine.
		scheduler.Next()
		waitFn()

		cancel()
		err := <-errCh
		require.NoError(t, err)
	}))

	t.Run("Successfully connects and receives a series of actions", withGateway(agentInfo, settings, func(
		t *testing.T,
		gateway gateway.FleetGateway,
		client *testingClient,
		dispatcher *testingDispatcher,
		scheduler *scheduler.Stepper,
	) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				// TODO: assert no events
				resp := wrapStrToResp(http.StatusOK, `
	{
		"actions": [
			{
				"type": "POLICY_CHANGE",
				"id": "id1",
				"data": {
					"policy": {
						"id": "policy-id"
					}
				}
			},
			{
				"type": "ANOTHER_ACTION",
				"id": "id2"
			}
		]
	}
	`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Len(t, actions, 2)
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		scheduler.Next()
		waitFn()

		cancel()
		err := <-errCh
		require.NoError(t, err)
	}))

	// Test the normal time based execution.
	t.Run("Periodically communicates with Fleet", func(t *testing.T) {
		scheduler := scheduler.NewPeriodic(150 * time.Millisecond)
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log, _ := logger.New("tst", false)
		stateStore := newStateStore(t, log)

		queue := &mockQueue{}
		queue.On("DequeueActions").Return([]fleetapi.Action{})
		queue.On("Actions").Return([]fleetapi.Action{})

		gateway, err := newFleetGatewayWithScheduler(
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)
		require.NoError(t, err)

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		func() {
			var count int
			for {
				waitFn()
				count++
				if count == 4 {
					return
				}
			}
		}()

		cancel()
		err = <-errCh
		require.NoError(t, err)
	})

	t.Run("queue action from checkin", func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log, _ := logger.New("tst", false)
		stateStore := newStateStore(t, log)

		ts := time.Now().UTC().Round(time.Second)
		queue := &mockQueue{}
		queue.On("Add", mock.Anything, ts.Add(time.Hour).Unix()).Return().Once()
		queue.On("DequeueActions").Return([]fleetapi.Action{})
		queue.On("Actions").Return([]fleetapi.Action{})

		gateway, err := newFleetGatewayWithScheduler(
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)
		require.NoError(t, err)

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, fmt.Sprintf(`{"actions": [{
						"type": "UPGRADE",
						"id": "id1",
						"start_time": "%s",
						"expiration": "%s",
						"data": {
							"version": "1.2.3"
						}
					}]}`,
					ts.Add(time.Hour).Format(time.RFC3339),
					ts.Add(2*time.Hour).Format(time.RFC3339),
				))
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		scheduler.Next()
		waitFn()
		queue.AssertExpectations(t)

		cancel()
		err = <-errCh
		require.NoError(t, err)
	})

	t.Run("run action from queue", func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log, _ := logger.New("tst", false)
		stateStore := newStateStore(t, log)

		ts := time.Now().UTC().Round(time.Second)
		queue := &mockQueue{}
		queue.On("DequeueActions").Return([]fleetapi.Action{&fleetapi.ActionUpgrade{ActionID: "id1", ActionType: "UPGRADE", ActionStartTime: ts.Add(-1 * time.Hour).Format(time.RFC3339), ActionExpiration: ts.Add(time.Hour).Format(time.RFC3339)}}).Once()
		queue.On("Actions").Return([]fleetapi.Action{})

		gateway, err := newFleetGatewayWithScheduler(
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)
		require.NoError(t, err)

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, `{"actions": []}`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 1, len(actions))
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		scheduler.Next()
		waitFn()
		queue.AssertExpectations(t)

		cancel()
		err = <-errCh
		require.NoError(t, err)
	})

	t.Run("discard expired action from queue", func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log, _ := logger.New("tst", false)
		stateStore := newStateStore(t, log)

		ts := time.Now().UTC().Round(time.Second)
		queue := &mockQueue{}
		queue.On("DequeueActions").Return([]fleetapi.Action{&fleetapi.ActionUpgrade{ActionID: "id1", ActionType: "UPGRADE", ActionStartTime: ts.Add(-2 * time.Hour).Format(time.RFC3339), ActionExpiration: ts.Add(-1 * time.Hour).Format(time.RFC3339)}}).Once()
		queue.On("Actions").Return([]fleetapi.Action{})

		gateway, err := newFleetGatewayWithScheduler(
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)
		require.NoError(t, err)

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, `{"actions": []}`)
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				require.Equal(t, 0, len(actions))
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		scheduler.Next()
		waitFn()
		queue.AssertExpectations(t)

		cancel()
		err = <-errCh
		require.NoError(t, err)
	})

	t.Run("cancel action from checkin", func(t *testing.T) {
		scheduler := scheduler.NewStepper()
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log, _ := logger.New("tst", false)
		stateStore := newStateStore(t, log)

		ts := time.Now().UTC().Round(time.Second)
		queue := &mockQueue{}
		queue.On("Add", mock.Anything, ts.Add(-1*time.Hour).Unix()).Return().Once()
		queue.On("DequeueActions").Return([]fleetapi.Action{})
		queue.On("Actions").Return([]fleetapi.Action{}).Maybe() // this test seems flakey if we check for this call
		// queue.Cancel does not need to be mocked here as it is ran in the cancel action dispatcher.

		gateway, err := newFleetGatewayWithScheduler(
			log,
			settings,
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)
		require.NoError(t, err)

		waitFn := ackSeq(
			client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
				resp := wrapStrToResp(http.StatusOK, fmt.Sprintf(`{"actions": [{
						"type": "UPGRADE",
						"id": "id1",
						"start_time": "%s",
						"expiration": "%s",
						"data": {
							"version": "1.2.3"
						}
					}, {
						"type": "CANCEL",
						"id": "id2",
						"data": {
							"target_id": "id1"
						}
					}]}`,
					ts.Add(-1*time.Hour).Format(time.RFC3339),
					ts.Add(2*time.Hour).Format(time.RFC3339),
				))
				return resp, nil
			}),
			dispatcher.Answer(func(actions ...fleetapi.Action) error {
				return nil
			}),
		)

		errCh := runFleetGateway(ctx, gateway)

		scheduler.Next()
		waitFn()
		queue.AssertExpectations(t)

		cancel()
		err = <-errCh
		require.NoError(t, err)
	})

	t.Run("Test the wait loop is interruptible", func(t *testing.T) {
		// 20mins is the double of the base timeout values for golang test suites.
		// If we cannot interrupt we will timeout.
		d := 20 * time.Minute
		scheduler := scheduler.NewPeriodic(d)
		client := newTestingClient()
		dispatcher := newTestingDispatcher()

		ctx, cancel := context.WithCancel(context.Background())

		log, _ := logger.New("tst", false)
		stateStore := newStateStore(t, log)

		queue := &mockQueue{}
		queue.On("DequeueActions").Return([]fleetapi.Action{})
		queue.On("Actions").Return([]fleetapi.Action{})

		gateway, err := newFleetGatewayWithScheduler(
			log,
			&fleetGatewaySettings{
				Duration: d,
				Backoff:  backoffSettings{Init: 1 * time.Second, Max: 30 * time.Second},
			},
			agentInfo,
			client,
			dispatcher,
			scheduler,
			noop.New(),
			&emptyStateFetcher{},
			stateStore,
			queue,
		)
		require.NoError(t, err)

		ch1 := dispatcher.Answer(func(actions ...fleetapi.Action) error { return nil })
		ch2 := client.Answer(func(headers http.Header, body io.Reader) (*http.Response, error) {
			resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
			return resp, nil
		})

		errCh := runFleetGateway(ctx, gateway)

		// Silently dispatch action.
		go func() {
			for range ch1 {
			}
		}()

		// Make sure that all API calls to the checkin API are successful, the following will happen:

		// block on the first call.
		<-ch2

		go func() {
			// drain the channel
			for range ch2 {
			}
		}()

		// 1. Gateway will check the API on boot.
		// 2. WaitTick() will block for 20 minutes.
		// 3. Stop will should unblock the wait.
		cancel()
		err = <-errCh
		require.NoError(t, err)
	})

}

func TestRetriesOnFailures(t *testing.T) {
	agentInfo := &testAgentInfo{}
	settings := &fleetGatewaySettings{
		Duration: 5 * time.Second,
		Backoff:  backoffSettings{Init: 100 * time.Millisecond, Max: 5 * time.Second},
	}

	t.Run("When the gateway fails to communicate with the checkin API we will retry",
		withGateway(agentInfo, settings, func(
			t *testing.T,
			gateway gateway.FleetGateway,
			client *testingClient,
			dispatcher *testingDispatcher,
			scheduler *scheduler.Stepper,
		) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fail := func(_ http.Header, _ io.Reader) (*http.Response, error) {
				return wrapStrToResp(http.StatusInternalServerError, "something is bad"), nil
			}
			clientWaitFn := client.Answer(fail)

			errCh := runFleetGateway(ctx, gateway)

			// Initial tick is done out of bound so we can block on channels.
			scheduler.Next()

			// Simulate a 500 errors for the next 3 calls.
			<-clientWaitFn
			<-clientWaitFn
			<-clientWaitFn

			// API recover
			waitFn := ackSeq(
				client.Answer(func(_ http.Header, body io.Reader) (*http.Response, error) {
					resp := wrapStrToResp(http.StatusOK, `{ "actions": [] }`)
					return resp, nil
				}),

				dispatcher.Answer(func(actions ...fleetapi.Action) error {
					require.Equal(t, 0, len(actions))
					return nil
				}),
			)

			waitFn()

			cancel()
			err := <-errCh
			require.NoError(t, err)
		}))

	t.Run("The retry loop is interruptible",
		withGateway(agentInfo, &fleetGatewaySettings{
			Duration: 0 * time.Second,
			Backoff:  backoffSettings{Init: 10 * time.Minute, Max: 20 * time.Minute},
		}, func(
			t *testing.T,
			gateway gateway.FleetGateway,
			client *testingClient,
			dispatcher *testingDispatcher,
			scheduler *scheduler.Stepper,
		) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fail := func(_ http.Header, _ io.Reader) (*http.Response, error) {
				return wrapStrToResp(http.StatusInternalServerError, "something is bad"), nil
			}
			waitChan := client.Answer(fail)

			errCh := runFleetGateway(ctx, gateway)

			// Initial tick is done out of bound so we can block on channels.
			scheduler.Next()

			// Fail to enter retry loop, all other calls will fails and will force to wait on big initial
			// delay.
			<-waitChan

			cancel()
			err := <-errCh
			require.NoError(t, err)
		}))
}

type testAgentInfo struct{}

func (testAgentInfo) AgentID() string { return "agent-secret" }

type emptyStateFetcher struct{}

func (e *emptyStateFetcher) State() coordinator.State {
	return coordinator.State{}
}

func runFleetGateway(ctx context.Context, g gateway.FleetGateway) <-chan error {
	done := make(chan bool)
	errCh := make(chan error, 1)
	go func() {
		err := g.Run(ctx)
		close(done)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-g.Errors():
				// ignore errors here
			}
		}
	}()
	return errCh
}

func newStateStore(t *testing.T, log *logger.Logger) *store.StateStore {
	dir, err := ioutil.TempDir("", "fleet-gateway-unit-test")
	require.NoError(t, err)

	filename := filepath.Join(dir, "state.enc")
	diskStore := storage.NewDiskStore(filename)
	stateStore, err := store.NewStateStore(log, diskStore)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return stateStore
}
