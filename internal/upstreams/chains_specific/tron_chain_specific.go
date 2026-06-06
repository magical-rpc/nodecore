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
	"github.com/drpcorg/nodecore/pkg/chains"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var errTronSubscriptionsUnsupported = errors.New("tron head subscriptions are not supported")

type TronChainSpecificObject struct {
	ctx             context.Context
	upstreamId      string
	connector       connectors.ApiConnector
	configuredChain *chains.ConfiguredChain
	options         *chains.Options
}

func (t *TronChainSpecificObject) LabelsProcessor() labels.LabelsProcessor {
	return nil
}

func (t *TronChainSpecificObject) LowerBoundProcessor() lower_bounds.LowerBoundProcessor {
	return nil
}

func (t *TronChainSpecificObject) HealthValidators() []validations.Validator[protocol.AvailabilityStatus] {
	validators := make([]validations.Validator[protocol.AvailabilityStatus], 0)

	if *t.options.ValidateSyncing {
		validators = append(validators, validations.NewEthSyncingValidator(t.upstreamId, t.configuredChain, t.connector, t.options.InternalTimeout))
	}
	if *t.options.ValidatePeers {
		validators = append(validators, validations.NewEthPeersValidator(t.upstreamId, t.configuredChain.Chain, t.connector, t.options))
	}

	return validators
}

func (t *TronChainSpecificObject) SettingsValidators() []validations.Validator[validations.ValidationSettingResult] {
	validators := make([]validations.Validator[validations.ValidationSettingResult], 0)

	if !*t.options.DisableChainValidation {
		validators = append(validators, validations.NewEthChainValidator(t.upstreamId, t.connector, t.configuredChain, t.options))
	}

	return validators
}

func (t *TronChainSpecificObject) GetLatestBlock(ctx context.Context) (protocol.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, t.options.InternalTimeout)
	defer cancel()

	request, err := protocol.NewInternalUpstreamJsonRpcRequest("eth_blockNumber", nil, t.configuredChain.Chain)
	if err != nil {
		return protocol.ZeroBlock{}, err
	}

	response := t.connector.SendRequest(ctx, request)
	if response.HasError() {
		return protocol.ZeroBlock{}, response.GetError()
	}

	return t.ParseBlock(response.ResponseResult())
}

func (t *TronChainSpecificObject) GetFinalizedBlock(context.Context) (protocol.Block, error) {
	return protocol.ZeroBlock{}, nil
}

func (t *TronChainSpecificObject) ParseBlock(blockBytes []byte) (protocol.Block, error) {
	var rawHeight string
	if err := sonic.Unmarshal(blockBytes, &rawHeight); err != nil {
		return protocol.ZeroBlock{}, fmt.Errorf("couldn't parse the tron block number, reason - %s", err.Error())
	}

	height, err := hexutil.DecodeUint64(rawHeight)
	if err != nil {
		return protocol.ZeroBlock{}, fmt.Errorf("couldn't decode the tron block number '%s', reason - %s", rawHeight, err.Error())
	}

	return protocol.NewBlockWithHeight(height), nil
}

func (t *TronChainSpecificObject) ParseSubscriptionBlock([]byte) (protocol.Block, error) {
	return protocol.ZeroBlock{}, errTronSubscriptionsUnsupported
}

func (t *TronChainSpecificObject) SubscribeHeadRequest() (protocol.RequestHolder, error) {
	return nil, errTronSubscriptionsUnsupported
}

func NewTronChainSpecificObject(
	ctx context.Context,
	configuredChain *chains.ConfiguredChain,
	upstreamId string,
	connector connectors.ApiConnector,
	options *chains.Options,
) *TronChainSpecificObject {
	return &TronChainSpecificObject{
		ctx:             ctx,
		upstreamId:      upstreamId,
		connector:       connector,
		configuredChain: configuredChain,
		options:         options,
	}
}

var _ ChainSpecific = (*TronChainSpecificObject)(nil)
