package chains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChainByGrpcId(t *testing.T) {
	ethereum := GetChain("ethereum")
	assert.NotNil(t, ethereum)
	assert.NotEqual(t, UnknownChain, ethereum)

	byGrpc := GetChainByGrpcId(ethereum.GrpcId)
	assert.Equal(t, ethereum.Chain, byGrpc.Chain)
	assert.Equal(t, ethereum.MethodSpec, byGrpc.MethodSpec)
	assert.Equal(t, ethereum.ShortNames[0], byGrpc.ShortNames[0])
}

func TestGetChainByGrpcIdUnknown(t *testing.T) {
	unknown := GetChainByGrpcId(-1)
	assert.Equal(t, UnknownChain, unknown)
}

func TestBitcoinAndTronMethodSpecs(t *testing.T) {
	bitcoin := GetChain("bitcoin")
	assert.NotEqual(t, UnknownChain, bitcoin)
	assert.Equal(t, Bitcoin, bitcoin.Type)
	assert.Equal(t, "bitcoin", bitcoin.MethodSpec)

	tron := GetChain("tron")
	assert.NotEqual(t, UnknownChain, tron)
	assert.Equal(t, Ethereum, tron.Type)
	assert.Equal(t, "tron", tron.MethodSpec)
}
