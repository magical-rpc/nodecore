package validations

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/drpcorg/nodecore/internal/protocol"
	"github.com/drpcorg/nodecore/internal/upstreams/connectors"
	"github.com/drpcorg/nodecore/pkg/chains"
	"github.com/rs/zerolog/log"
)

type BitcoinSyncingValidator struct {
	upstreamId      string
	chain           *chains.ConfiguredChain
	connector       connectors.ApiConnector
	internalTimeout time.Duration
}

func NewBitcoinSyncingValidator(
	upstreamId string,
	chain *chains.ConfiguredChain,
	connector connectors.ApiConnector,
	internalTimeout time.Duration,
) *BitcoinSyncingValidator {
	return &BitcoinSyncingValidator{
		upstreamId:      upstreamId,
		chain:           chain,
		connector:       connector,
		internalTimeout: internalTimeout,
	}
}

func (b *BitcoinSyncingValidator) Validate() protocol.AvailabilityStatus {
	info, err := b.getBlockchainInfo()
	if err != nil {
		log.Error().Err(err).Msgf("unable to get bitcoin syncing status of upstream '%s'", b.upstreamId)
		return protocol.Unavailable
	}

	if info.InitialBlockDownload {
		return protocol.Syncing
	}
	if info.Headers > info.Blocks && int64(info.Headers-info.Blocks) > b.chain.Settings.Lags.Syncing {
		return protocol.Syncing
	}
	return protocol.Available
}

func (b *BitcoinSyncingValidator) getBlockchainInfo() (*bitcoinBlockchainInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.internalTimeout)
	defer cancel()

	request, err := protocol.NewInternalUpstreamJsonRpcRequest("getblockchaininfo", nil, b.chain.Chain)
	if err != nil {
		return nil, err
	}

	response := b.connector.SendRequest(ctx, request)
	return parseBitcoinBlockchainInfo(response)
}

type BitcoinPeersValidator struct {
	upstreamId string
	chain      chains.Chain
	connector  connectors.ApiConnector
	options    *chains.Options
}

func NewBitcoinPeersValidator(
	upstreamId string,
	chain chains.Chain,
	connector connectors.ApiConnector,
	options *chains.Options,
) *BitcoinPeersValidator {
	return &BitcoinPeersValidator{
		upstreamId: upstreamId,
		chain:      chain,
		connector:  connector,
		options:    options,
	}
}

func (b *BitcoinPeersValidator) Validate() protocol.AvailabilityStatus {
	peers, err := b.getPeerCount()
	if err != nil {
		log.Error().Err(err).Msgf("unable to get bitcoin peer count of upstream '%s'", b.upstreamId)
		return protocol.Unavailable
	}
	if peers < b.options.MinPeers {
		return protocol.Immature
	}
	return protocol.Available
}

func (b *BitcoinPeersValidator) getPeerCount() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.options.InternalTimeout)
	defer cancel()

	peers, err := b.getConnectionCount(ctx)
	if err == nil {
		return peers, nil
	}

	return b.getNetworkInfoConnectionCount(ctx)
}

func (b *BitcoinPeersValidator) getConnectionCount(ctx context.Context) (int64, error) {
	request, err := protocol.NewInternalUpstreamJsonRpcRequest("getconnectioncount", nil, b.chain)
	if err != nil {
		return 0, err
	}

	response := b.connector.SendRequest(ctx, request)
	if response.HasError() {
		return 0, response.GetError()
	}

	var peers int64
	if err := sonic.Unmarshal(response.ResponseResult(), &peers); err != nil {
		return 0, err
	}

	return peers, nil
}

func (b *BitcoinPeersValidator) getNetworkInfoConnectionCount(ctx context.Context) (int64, error) {
	request, err := protocol.NewInternalUpstreamJsonRpcRequest("getnetworkinfo", nil, b.chain)
	if err != nil {
		return 0, err
	}

	response := b.connector.SendRequest(ctx, request)
	if response.HasError() {
		return 0, response.GetError()
	}

	networkInfo := struct {
		Connections int64 `json:"connections"`
	}{}
	if err := sonic.Unmarshal(response.ResponseResult(), &networkInfo); err != nil {
		return 0, err
	}

	return networkInfo.Connections, nil
}

type bitcoinBlockchainInfo struct {
	Chain                string `json:"chain"`
	Blocks               uint64 `json:"blocks"`
	Headers              uint64 `json:"headers"`
	BestBlockHash        string `json:"bestblockhash"`
	InitialBlockDownload bool   `json:"initialblockdownload"`
}

func parseBitcoinBlockchainInfo(response protocol.ResponseHolder) (*bitcoinBlockchainInfo, error) {
	if response.HasError() {
		return nil, response.GetError()
	}

	info := bitcoinBlockchainInfo{}
	if err := sonic.Unmarshal(response.ResponseResult(), &info); err != nil {
		return nil, fmt.Errorf("couldn't parse bitcoin blockchain info, reason - %s", err.Error())
	}

	return &info, nil
}

var _ HealthValidator = (*BitcoinSyncingValidator)(nil)
var _ HealthValidator = (*BitcoinPeersValidator)(nil)
