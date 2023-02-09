package evm_test

import (
	"context"
	"math/big"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/types"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

func (s *EVMTestSuite) TestSubmitDataCommitment() {
	// deploy a new bridge contract
	_, _, _, err := s.Client.DeployQGBContract(s.Chain.Auth, s.Chain.Backend, *s.InitVs, 1, true)
	s.NoError(err)

	// we just need something to sign over, it doesn't matter what
	commitment := ethcmn.HexToHash("0x12345")
	signBytes := types.DataCommitmentTupleRootSignBytes(
		big.NewInt(2),
		commitment[:],
	)

	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), s.VsPrivateKey)
	s.NoError(err)

	evmVals := make([]wrapper.Validator, len(s.InitVs.Members))
	for i, val := range s.InitVs.Members {
		evmVals[i] = wrapper.Validator{
			Addr:  ethcmn.HexToAddress(val.EvmAddress),
			Power: big.NewInt(int64(val.Power)),
		}
	}

	hexSig := ethcmn.Bytes2Hex(signature)
	v, r, ss := evm.SigToVRS(hexSig)
	tx, err := s.Client.SubmitDataRootTupleRoot(
		s.Chain.Auth,
		commitment,
		2,
		*s.InitVs,
		[]wrapper.Signature{
			{
				V: v,
				R: r,
				S: ss,
			},
		},
	)
	s.NoError(err)
	s.Chain.Backend.Commit()

	recp, err := s.Chain.Backend.TransactionReceipt(context.TODO(), tx.Hash())
	s.NoError(err)
	s.Assert().Equal(uint64(1), recp.Status)

	dcNonce, err := s.Client.StateLastEventNonce(nil)
	s.NoError(err)
	s.Assert().Equal(uint64(2), dcNonce)
}

func (s *EVMTestSuite) TestUpdateValset() {
	// deploy a new bridge contract
	_, _, _, err := s.Client.DeployQGBContract(s.Chain.Auth, s.Chain.Backend, *s.InitVs, 1, true)
	s.NoError(err)

	updatedValset := celestiatypes.Valset{
		Members: []celestiatypes.BridgeValidator{
			{
				EvmAddress: "0x9c2B12b5a07FC6D719Ed7646e5041A7E85758328",
				Power:      5000,
			},
			{
				EvmAddress: "0x9c2B12b5a07FC6D719Ed7646e5041A7E85758327",
				Power:      5000,
			},
		},
		// because the bridge was redeployed
		Nonce:  2,
		Height: 10,
	}

	signBytes, err := updatedValset.SignBytes()
	s.NoError(err)
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), s.VsPrivateKey)
	s.NoError(err)

	hexSig := ethcmn.Bytes2Hex(signature)

	evmVals := make([]wrapper.Validator, len(s.InitVs.Members))
	for i, val := range s.InitVs.Members {
		evmVals[i] = wrapper.Validator{
			Addr:  ethcmn.HexToAddress(val.EvmAddress),
			Power: big.NewInt(int64(val.Power)),
		}
	}

	thresh := updatedValset.TwoThirdsThreshold()

	v, r, ss := evm.SigToVRS(hexSig)

	tx, err := s.Client.UpdateValidatorSet(
		s.Chain.Auth,
		2,
		thresh,
		*s.InitVs,
		updatedValset,
		[]wrapper.Signature{
			{
				V: v,
				R: r,
				S: ss,
			},
		},
	)
	s.NoError(err)
	s.Chain.Backend.Commit()

	recp, err := s.Chain.Backend.TransactionReceipt(context.TODO(), tx.Hash())
	s.NoError(err)
	s.Equal(uint64(1), recp.Status)

	nonce, err := s.Client.StateLastEventNonce(nil)
	s.NoError(err)
	// check that the validator set was changed.
	s.Equal(uint64(2), nonce)
}
