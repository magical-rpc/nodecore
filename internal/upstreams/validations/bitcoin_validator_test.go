package validations_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/drpcorg/nodecore/internal/protocol"
	"github.com/drpcorg/nodecore/internal/upstreams/validations"
	"github.com/drpcorg/nodecore/pkg/chains"
	"github.com/drpcorg/nodecore/pkg/test_utils/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBitcoinChainValidatorValidResult(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(connector, "getblockchaininfo", bitcoinBlockchainInfoResponse("main", 850000, 850000, false))

	validator := validations.NewBitcoinChainValidator("id", connector, chains.GetChain("bitcoin"), bitcoinValidationOptions())
	actualResult := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, validations.Valid, actualResult)
}

func TestBitcoinChainValidatorMismatchThenFatalError(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(connector, "getblockchaininfo", bitcoinBlockchainInfoResponse("test", 100, 100, false))

	validator := validations.NewBitcoinChainValidator("id", connector, chains.GetChain("bitcoin"), bitcoinValidationOptions())
	actualResult := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, validations.FatalSettingError, actualResult)
}

func TestBitcoinChainValidatorErrorThenSettingsError(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(
		connector,
		"getblockchaininfo",
		protocol.NewHttpUpstreamResponseWithError(protocol.ResponseErrorWithData(1, "rpc error", nil)),
	)

	validator := validations.NewBitcoinChainValidator("id", connector, chains.GetChain("bitcoin"), bitcoinValidationOptions())
	actualResult := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, validations.SettingsError, actualResult)
}

func TestBitcoinSyncingValidatorInitialBlockDownload(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(connector, "getblockchaininfo", bitcoinBlockchainInfoResponse("main", 100, 1000, true))

	validator := validations.NewBitcoinSyncingValidator("id", chains.GetChain("bitcoin"), connector, time.Second)
	actualStatus := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, protocol.Syncing, actualStatus)
}

func TestBitcoinSyncingValidatorHeadersLag(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(connector, "getblockchaininfo", bitcoinBlockchainInfoResponse("main", 100, 110, false))

	validator := validations.NewBitcoinSyncingValidator("id", chains.GetChain("bitcoin"), connector, time.Second)
	actualStatus := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, protocol.Syncing, actualStatus)
}

func TestBitcoinSyncingValidatorAvailable(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(connector, "getblockchaininfo", bitcoinBlockchainInfoResponse("main", 100, 100, false))

	validator := validations.NewBitcoinSyncingValidator("id", chains.GetChain("bitcoin"), connector, time.Second)
	actualStatus := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, protocol.Available, actualStatus)
}

func TestBitcoinPeersValidatorImmature(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(connector, "getconnectioncount", protocol.NewSimpleHttpUpstreamResponse("1", []byte(`2`), protocol.JsonRpc))

	validator := validations.NewBitcoinPeersValidator("id", chains.GetChain("bitcoin").Chain, connector, bitcoinValidationOptions())
	actualStatus := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, protocol.Immature, actualStatus)
}

func TestBitcoinPeersValidatorFallbackNetworkInfo(t *testing.T) {
	connector := mocks.NewConnectorMock()
	expectBitcoinRequest(
		connector,
		"getconnectioncount",
		protocol.NewHttpUpstreamResponseWithError(protocol.ResponseErrorWithData(1, "rpc error", nil)),
	)
	expectBitcoinRequest(
		connector,
		"getnetworkinfo",
		protocol.NewSimpleHttpUpstreamResponse("1", []byte(`{"connections":5}`), protocol.JsonRpc),
	)

	validator := validations.NewBitcoinPeersValidator("id", chains.GetChain("bitcoin").Chain, connector, bitcoinValidationOptions())
	actualStatus := validator.Validate()

	connector.AssertExpectations(t)
	assert.Equal(t, protocol.Available, actualStatus)
}

func bitcoinValidationOptions() *chains.Options {
	return &chains.Options{
		InternalTimeout: time.Second,
		MinPeers:        3,
	}
}

func bitcoinBlockchainInfoResponse(chain string, blocks, headers uint64, initialBlockDownload bool) protocol.ResponseHolder {
	return protocol.NewSimpleHttpUpstreamResponse(
		"1",
		[]byte(`{"chain":"`+chain+`","blocks":`+uintToString(blocks)+`,"headers":`+uintToString(headers)+`,"bestblockhash":"00000000000000000000aabbccddeeff00112233445566778899aabbccddeeff","initialblockdownload":`+boolToString(initialBlockDownload)+`}`),
		protocol.JsonRpc,
	)
}

func expectBitcoinRequest(connector *mocks.ConnectorMock, method string, response protocol.ResponseHolder) {
	call := connector.
		On("SendRequest", mock.Anything, mock.MatchedBy(func(request protocol.RequestHolder) bool {
			return request.Method() == method && request.RequestType() == protocol.JsonRpc
		})).
		Return(response)

	if response.HasError() && method == "getblockchaininfo" {
		call.Times(validations.RetryMaxAttempts)
		return
	}

	call.Once()
}

func uintToString(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func boolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
