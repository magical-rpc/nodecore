package specific

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/drpcorg/nodecore/internal/protocol"
	"github.com/drpcorg/nodecore/internal/upstreams/connectors"
	"github.com/drpcorg/nodecore/internal/upstreams/labels"
	"github.com/drpcorg/nodecore/internal/upstreams/lower_bounds"
	"github.com/drpcorg/nodecore/internal/upstreams/validations"
	"github.com/drpcorg/nodecore/pkg/blockchain"
	"github.com/drpcorg/nodecore/pkg/chains"
)

var errBitcoinSubscriptionsUnsupported = errors.New("bitcoin head subscriptions are not supported")

type BitcoinChainSpecificObject struct {
	ctx             context.Context
	upstreamId      string
	configuredChain *chains.ConfiguredChain
	connector       connectors.ApiConnector
	options         *chains.Options
}

func (b *BitcoinChainSpecificObject) LabelsProcessor() labels.LabelsProcessor {
	return nil
}

func (b *BitcoinChainSpecificObject) LowerBoundProcessor() lower_bounds.LowerBoundProcessor {
	return nil
}

func (b *BitcoinChainSpecificObject) HealthValidators() []validations.Validator[protocol.AvailabilityStatus] {
	validators := make([]validations.Validator[protocol.AvailabilityStatus], 0)

	if *b.options.ValidateSyncing {
		validators = append(validators, validations.NewBitcoinSyncingValidator(b.upstreamId, b.configuredChain, b.connector, b.options.InternalTimeout))
	}
	if *b.options.ValidatePeers {
		validators = append(validators, validations.NewBitcoinPeersValidator(b.upstreamId, b.configuredChain.Chain, b.connector, b.options))
	}

	return validators
}

func (b *BitcoinChainSpecificObject) SettingsValidators() []validations.Validator[validations.ValidationSettingResult] {
	validators := make([]validations.Validator[validations.ValidationSettingResult], 0)

	if !*b.options.DisableChainValidation {
		validators = append(validators, validations.NewBitcoinChainValidator(b.upstreamId, b.connector, b.configuredChain, b.options))
	}

	return validators
}

func (b *BitcoinChainSpecificObject) GetLatestBlock(ctx context.Context) (protocol.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, b.options.InternalTimeout)
	defer cancel()

	request, err := protocol.NewInternalUpstreamJsonRpcRequest("getblockchaininfo", nil, b.configuredChain.Chain)
	if err != nil {
		return protocol.ZeroBlock{}, err
	}

	response := b.connector.SendRequest(ctx, request)
	if response.HasError() {
		return protocol.ZeroBlock{}, response.GetError()
	}

	return b.ParseBlock(response.ResponseResult())
}

func (b *BitcoinChainSpecificObject) GetFinalizedBlock(context.Context) (protocol.Block, error) {
	return protocol.ZeroBlock{}, nil
}

func (b *BitcoinChainSpecificObject) ParseBlock(blockBytes []byte) (protocol.Block, error) {
	info := BitcoinBlockchainInfo{}
	if err := sonic.Unmarshal(blockBytes, &info); err != nil {
		return protocol.ZeroBlock{}, fmt.Errorf("couldn't parse the bitcoin block, reason - %s", err.Error())
	}
	if info.BestBlockHash == "" {
		return protocol.ZeroBlock{}, fmt.Errorf("couldn't parse the bitcoin block, got '%s'", string(blockBytes))
	}

	return protocol.NewBlock(
		info.Blocks,
		0,
		blockchain.NewHashIdFromString(info.BestBlockHash),
		blockchain.EmptyHash,
	), nil
}

func (b *BitcoinChainSpecificObject) ParseSubscriptionBlock([]byte) (protocol.Block, error) {
	return protocol.ZeroBlock{}, errBitcoinSubscriptionsUnsupported
}

func (b *BitcoinChainSpecificObject) SubscribeHeadRequest() (protocol.RequestHolder, error) {
	return nil, errBitcoinSubscriptionsUnsupported
}

func NewBitcoinChainSpecificObject(
	ctx context.Context,
	configuredChain *chains.ConfiguredChain,
	upstreamId string,
	connector connectors.ApiConnector,
	options *chains.Options,
) *BitcoinChainSpecificObject {
	return &BitcoinChainSpecificObject{
		ctx:             ctx,
		upstreamId:      upstreamId,
		configuredChain: configuredChain,
		connector:       connector,
		options:         options,
	}
}

type BitcoinBlockchainInfo struct {
	Chain                string `json:"chain"`
	Blocks               uint64 `json:"blocks"`
	Headers              uint64 `json:"headers"`
	BestBlockHash        string `json:"bestblockhash"`
	InitialBlockDownload bool   `json:"initialblockdownload"`
}

var _ ChainSpecific = (*BitcoinChainSpecificObject)(nil)
