package validations

import (
	"context"
	"time"

	"github.com/drpcorg/nodecore/internal/protocol"
	"github.com/drpcorg/nodecore/internal/upstreams/connectors"
	"github.com/drpcorg/nodecore/pkg/chains"
	"github.com/failsafe-go/failsafe-go"
	"github.com/rs/zerolog/log"
)

type BitcoinChainValidator struct {
	upstreamId      string
	connector       connectors.ApiConnector
	chain           *chains.ConfiguredChain
	internalTimeout time.Duration

	executor failsafe.Executor[protocol.ResponseHolder]
}

func NewBitcoinChainValidator(
	upstreamId string,
	connector connectors.ApiConnector,
	chain *chains.ConfiguredChain,
	options *chains.Options,
) *BitcoinChainValidator {
	return &BitcoinChainValidator{
		upstreamId:      upstreamId,
		connector:       connector,
		chain:           chain,
		internalTimeout: options.InternalTimeout,
		executor:        validatorExecutor(upstreamId, "bitcoinChainValidation", nil),
	}
}

func (b *BitcoinChainValidator) Validate() ValidationSettingResult {
	ctx, cancel := context.WithTimeout(context.Background(), b.internalTimeout)
	defer cancel()

	info, err := b.getBlockchainInfo(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get bitcoin blockchain info of upstream '%s'", b.upstreamId)
		return SettingsError
	}

	expectedChains := expectedBitcoinNodeChains(b.chain)
	if len(expectedChains) == 0 || !containsString(expectedChains, info.Chain) {
		chainName := b.chain.Chain.String()
		if len(b.chain.ShortNames) > 0 {
			chainName = b.chain.ShortNames[0]
		}
		log.Error().Msgf(
			"'%s' is specified for upstream '%s', but bitcoin node reports chain '%s'",
			chainName,
			b.upstreamId,
			info.Chain,
		)
		return FatalSettingError
	}

	return Valid
}

func (b *BitcoinChainValidator) getBlockchainInfo(ctx context.Context) (*bitcoinBlockchainInfo, error) {
	request, err := protocol.NewInternalUpstreamJsonRpcRequest("getblockchaininfo", nil, b.chain.Chain)
	if err != nil {
		return nil, err
	}

	response, _ := b.executor.
		Get(func() (protocol.ResponseHolder, error) {
			return b.connector.SendRequest(ctx, request), nil
		})

	return parseBitcoinBlockchainInfo(response)
}

func expectedBitcoinNodeChains(chain *chains.ConfiguredChain) []string {
	if containsString(chain.ShortNames, "bitcoin-testnet") {
		return []string{"test", "testnet4"}
	}
	if containsString(chain.ShortNames, "bitcoin") || containsString(chain.ShortNames, "bitcoin-mainnet") {
		return []string{"main"}
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ SettingsValidator = (*BitcoinChainValidator)(nil)
