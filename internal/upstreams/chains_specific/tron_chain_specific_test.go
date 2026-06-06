package specific_test

import (
	"context"
	"testing"
	"time"

	"github.com/drpcorg/nodecore/internal/protocol"
	specific "github.com/drpcorg/nodecore/internal/upstreams/chains_specific"
	"github.com/drpcorg/nodecore/pkg/chains"
	"github.com/drpcorg/nodecore/pkg/test_utils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTronGetLatestBlock(t *testing.T) {
	connector := mocks.NewConnectorMock()
	response := protocol.NewHttpUpstreamResponse("1", []byte(`{"jsonrpc":"2.0","result":"0x42668a1"}`), 200, protocol.JsonRpc)

	connector.
		On("SendRequest", mock.Anything, mock.MatchedBy(jsonRpcMethodMatcher("eth_blockNumber"))).
		Return(response).
		Once()

	block, err := newTronChainSpecific(connector).GetLatestBlock(context.Background())
	assert.NoError(t, err)

	connector.AssertExpectations(t)
	assert.Equal(t, uint64(69626017), block.Height)
}

func TestTronHeadSubscriptionsUnsupported(t *testing.T) {
	chainSpecific := newTronChainSpecific(nil)

	req, err := chainSpecific.SubscribeHeadRequest()
	assert.Nil(t, req)
	assert.ErrorContains(t, err, "tron head subscriptions are not supported")

	block, err := chainSpecific.ParseSubscriptionBlock([]byte(`{}`))
	assert.True(t, block.IsFullEmpty())
	assert.ErrorContains(t, err, "tron head subscriptions are not supported")
}

func newTronChainSpecific(connector *mocks.ConnectorMock) *specific.TronChainSpecificObject {
	return specific.NewTronChainSpecificObject(
		context.Background(),
		chains.GetChain("tron"),
		"id",
		connector,
		&chains.Options{
			InternalTimeout:             time.Second,
			DisableChainValidation:      new(false),
			ValidateSyncing:             new(false),
			ValidatePeers:               new(false),
			DisableLowerBoundsDetection: new(false),
			DisableLabelsDetection:      new(false),
		},
	)
}
