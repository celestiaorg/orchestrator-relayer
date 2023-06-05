package evm_test

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EVMTestSuite struct {
	suite.Suite
	Chain        *qgbtesting.EVMChain
	Client       *evm.Client
	InitVs       *celestiatypes.Valset
	VsPrivateKey *ecdsa.PrivateKey
}

func (s *EVMTestSuite) SetupTest() {
	t := s.T()
	testPrivateKey, err := crypto.HexToECDSA("64a1d6f0e760a8d62b4afdde4096f16f51b401eaaecc915740f71770ea76a8ad")
	s.VsPrivateKey = testPrivateKey
	require.NoError(t, err)
	s.Chain = qgbtesting.NewEVMChain(testPrivateKey)

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(testPrivateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	s.Client = qgbtesting.NewEVMClient(ks, &acc)
	s.InitVs, err = celestiatypes.NewValset(
		1,
		10,
		celestiatypes.InternalBridgeValidators{{
			Power:      1000,
			EVMAddress: ethcmn.HexToAddress("0x9c2B12b5a07FC6D719Ed7646e5041A7E85758329"),
		}},
		time.Now(),
	)
	require.NoError(t, err)
}

func (s *EVMTestSuite) TearDown() {
	s.Chain.Close()
}

func TestEVMSuite(t *testing.T) {
	suite.Run(t, new(EVMTestSuite))
}
