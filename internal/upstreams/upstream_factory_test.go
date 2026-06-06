package upstreams_test

import (
	"context"
	"testing"
	"time"

	"github.com/drpcorg/nodecore/internal/config"
	"github.com/drpcorg/nodecore/internal/dimensions"
	"github.com/drpcorg/nodecore/internal/protocol"
	"github.com/drpcorg/nodecore/internal/ratelimiter"
	"github.com/drpcorg/nodecore/internal/resilience"
	"github.com/drpcorg/nodecore/internal/upstreams"
	specs "github.com/drpcorg/nodecore/pkg/methods"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopStatsService struct{}

func (noopStatsService) AddRequestResults([]protocol.RequestResult) {}

func TestCreateBitcoinUpstreamJsonRpcConfigSucceeds(t *testing.T) {
	require.NoError(t, specs.NewMethodSpecLoader().Load())

	upstream, err := upstreams.CreateUpstream(
		context.Background(),
		bitcoinUpstreamConfig(
			[]*config.ApiConnectorConfig{
				{
					Type: config.JsonRpc,
					Url:  "http://127.0.0.1:8332",
					Headers: map[string]string{
						"Authorization": "Basic dXNlcjpwYXNz",
					},
				},
			},
			config.JsonRpc,
		),
		dimensions.NewBaseDimensionTracker(),
		noopStatsService{},
		resilience.CreateUpstreamExecutor(),
		0,
		newRateLimitBudgetRegistry(t),
		"",
	)

	require.NoError(t, err)
	assert.NotNil(t, upstream)
}

func TestCreateBitcoinUpstreamWebsocketConnectorRejected(t *testing.T) {
	require.NoError(t, specs.NewMethodSpecLoader().Load())

	_, err := upstreams.CreateUpstream(
		context.Background(),
		bitcoinUpstreamConfig(
			[]*config.ApiConnectorConfig{
				{
					Type: config.JsonRpc,
					Url:  "http://127.0.0.1:8332",
				},
				{
					Type: config.Ws,
					Url:  "ws://127.0.0.1:8333",
				},
			},
			config.JsonRpc,
		),
		dimensions.NewBaseDimensionTracker(),
		noopStatsService{},
		resilience.CreateUpstreamExecutor(),
		0,
		newRateLimitBudgetRegistry(t),
		"",
	)

	assert.ErrorContains(t, err, "bitcoin upstreams support only 'json-rpc' connectors, got 'websocket'")
}

func bitcoinUpstreamConfig(connectors []*config.ApiConnectorConfig, headConnector config.ApiConnectorType) *config.Upstream {
	return &config.Upstream{
		Id:             "bitcoin-upstream",
		ChainName:      "bitcoin",
		Connectors:     connectors,
		HeadConnector:  headConnector,
		PollInterval:   time.Second,
		Methods:        &config.MethodsConfig{BanDuration: time.Minute},
		FailsafeConfig: &config.FailsafeConfig{},
		Options: testUpstreamOptions(
			withDisableValidation(true),
			withDisableLowerBoundsDetection(true),
			withDisableLabelsDetection(true),
		),
	}
}

func newRateLimitBudgetRegistry(t *testing.T) *ratelimiter.RateLimitBudgetRegistry {
	t.Helper()

	registry, err := ratelimiter.NewRateLimitBudgetRegistry(nil, nil)
	require.NoError(t, err)

	return registry
}
