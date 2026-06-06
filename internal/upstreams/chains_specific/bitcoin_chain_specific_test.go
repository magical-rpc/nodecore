package specific_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drpcorg/nodecore/internal/protocol"
	specific "github.com/drpcorg/nodecore/internal/upstreams/chains_specific"
	"github.com/drpcorg/nodecore/internal/upstreams/validations"
	"github.com/drpcorg/nodecore/pkg/blockchain"
	"github.com/drpcorg/nodecore/pkg/chains"
	"github.com/drpcorg/nodecore/pkg/test_utils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBitcoinValidators(t *testing.T) {
	options := bitcoinChainSpecificOptions()
	options.DisableChainValidation = new(true)
	options.ValidateSyncing = new(false)
	options.ValidatePeers = new(false)

	connector := mocks.NewConnectorMock()
	chainSpecific := specific.NewBitcoinChainSpecificObject(context.Background(), chains.GetChain("bitcoin"), "id", connector, options)
	assert.Len(t, chainSpecific.SettingsValidators(), 0)
	assert.Len(t, chainSpecific.HealthValidators(), 0)

	options.DisableChainValidation = new(false)
	options.ValidateSyncing = new(true)
	options.ValidatePeers = new(true)
	chainSpecific = specific.NewBitcoinChainSpecificObject(context.Background(), chains.GetChain("bitcoin"), "id", connector, options)

	settingsValidators := chainSpecific.SettingsValidators()
	assert.Len(t, settingsValidators, 1)
	assert.IsType(t, &validations.BitcoinChainValidator{}, settingsValidators[0])

	healthValidators := chainSpecific.HealthValidators()
	assert.Len(t, healthValidators, 2)
	assert.IsType(t, &validations.BitcoinSyncingValidator{}, healthValidators[0])
	assert.IsType(t, &validations.BitcoinPeersValidator{}, healthValidators[1])
}

func TestBitcoinParseBlock(t *testing.T) {
	body := []byte(`{
	  "chain": "main",
	  "blocks": 850000,
	  "headers": 850000,
	  "bestblockhash": "00000000000000000000aabbccddeeff00112233445566778899aabbccddeeff",
	  "initialblockdownload": false
	}`)

	block, err := newBitcoinChainSpecific(nil).ParseBlock(body)
	assert.NoError(t, err)

	expected := protocol.NewBlock(
		850000,
		0,
		blockchain.NewHashIdFromString("00000000000000000000aabbccddeeff00112233445566778899aabbccddeeff"),
		blockchain.EmptyHash,
	)
	assert.Equal(t, expected, block)
}

func TestBitcoinGetLatestBlock(t *testing.T) {
	connector := mocks.NewConnectorMock()
	response := protocol.NewHttpUpstreamResponse("1", []byte(`{
	  "jsonrpc": "2.0",
	  "result": {
	    "chain": "main",
	    "blocks": 850000,
	    "headers": 850000,
	    "bestblockhash": "00000000000000000000aabbccddeeff00112233445566778899aabbccddeeff",
	    "initialblockdownload": false
	  }
	}`), 200, protocol.JsonRpc)

	connector.
		On("SendRequest", mock.Anything, mock.MatchedBy(jsonRpcMethodMatcher("getblockchaininfo"))).
		Return(response).
		Once()

	block, err := newBitcoinChainSpecific(connector).GetLatestBlock(context.Background())
	assert.NoError(t, err)

	connector.AssertExpectations(t)
	assert.Equal(t, uint64(850000), block.Height)
	assert.Equal(t, blockchain.NewHashIdFromString("00000000000000000000aabbccddeeff00112233445566778899aabbccddeeff"), block.Hash)
}

func TestBitcoinGetLatestBlockWithError(t *testing.T) {
	connector := mocks.NewConnectorMock()
	response := protocol.NewHttpUpstreamResponseWithError(protocol.ResponseErrorWithData(1, "block error", nil))

	connector.
		On("SendRequest", mock.Anything, mock.MatchedBy(jsonRpcMethodMatcher("getblockchaininfo"))).
		Return(response).
		Once()

	block, err := newBitcoinChainSpecific(connector).GetLatestBlock(context.Background())

	connector.AssertExpectations(t)
	assert.True(t, block.IsFullEmpty())

	var upErr *protocol.ResponseError
	assert.True(t, errors.As(err, &upErr))
	assert.Equal(t, 1, upErr.Code)
	assert.Equal(t, "block error", upErr.Message)
}

func TestBitcoinHeadSubscriptionsUnsupported(t *testing.T) {
	chainSpecific := newBitcoinChainSpecific(nil)

	req, err := chainSpecific.SubscribeHeadRequest()
	assert.Nil(t, req)
	assert.ErrorContains(t, err, "bitcoin head subscriptions are not supported")

	block, err := chainSpecific.ParseSubscriptionBlock([]byte(`{}`))
	assert.True(t, block.IsFullEmpty())
	assert.ErrorContains(t, err, "bitcoin head subscriptions are not supported")
}

func newBitcoinChainSpecific(connector *mocks.ConnectorMock) *specific.BitcoinChainSpecificObject {
	return specific.NewBitcoinChainSpecificObject(
		context.Background(),
		chains.GetChain("bitcoin"),
		"id",
		connector,
		bitcoinChainSpecificOptions(),
	)
}

func bitcoinChainSpecificOptions() *chains.Options {
	return &chains.Options{
		InternalTimeout:             time.Second,
		ValidationInterval:          time.Second,
		DisableValidation:           new(false),
		DisableSettingsValidation:   new(false),
		DisableChainValidation:      new(false),
		DisableHealthValidation:     new(false),
		DisableLowerBoundsDetection: new(false),
		DisableLabelsDetection:      new(false),
		ValidateSyncing:             new(true),
		ValidatePeers:               new(true),
		MinPeers:                    1,
	}
}

func jsonRpcMethodMatcher(method string) func(protocol.RequestHolder) bool {
	return func(request protocol.RequestHolder) bool {
		return request.Method() == method && request.RequestType() == protocol.JsonRpc
	}
}
