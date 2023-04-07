package p2p

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValsetConfirmValidate(t *testing.T) {
	validator := ValsetConfirmValidator{}
	signBytes := common.HexToHash("1234")

	evmAddress := "0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488"
	privateKey, _ := ethcrypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	tests := []struct {
		name    string
		key     string
		value   []byte
		wantErr bool
	}{
		{
			name: "valid valset confirm",
			key:  "/vc/b:" + evmAddress + ":" + signBytes.Hex(),
			value: func() []byte {
				signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
				require.NoError(t, err)
				vsc, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress(evmAddress),
					hex.EncodeToString(signature),
				))
				return vsc
			}(),
			wantErr: false,
		},
		{
			name:    "invalid key format",
			key:     "/vc/b/0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value:   nil,
			wantErr: true,
		},
		{
			name:    "invalid key namespace",
			key:     "/vcc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value:   nil,
			wantErr: true,
		},
		{
			name:    "short key evm address",
			key:     "/vc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value:   nil,
			wantErr: true,
		},
		{
			name: "not the same evm address in key and in confirm",
			key:  "/vc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value: func() []byte {
				vsc, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				return vsc
			}(),
			wantErr: true,
		},
		{
			name: "invalid signature",
			key:  "/vc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value: func() []byte {
				vsc, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031",
				))
				return vsc
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.key, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValsetConfirmSelect(t *testing.T) {
	validator := ValsetConfirmValidator{}
	signBytes := common.HexToHash("1234")
	evmAddress := "0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488"
	privateKey, _ := ethcrypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	tests := []struct {
		name          string
		key           string
		values        [][]byte
		expectedIndex int
		wantErr       bool
	}{
		{
			name: "first valset confirm is valid",
			key:  "/vc/b:" + evmAddress + ":" + signBytes.Hex(),
			values: func() [][]byte {
				signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
				require.NoError(t, err)
				vc1, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress(evmAddress),
					hex.EncodeToString(signature),
				))
				vc2, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				vc3, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			expectedIndex: 0,
			wantErr:       false,
		},
		{
			name: "second valset confirm is valid",
			key:  "/vc/b:" + evmAddress + ":" + signBytes.Hex(),
			values: func() [][]byte {
				vc1, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
				require.NoError(t, err)
				vc2, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress(evmAddress),
					hex.EncodeToString(signature),
				))
				vc3, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			expectedIndex: 1,
			wantErr:       false,
		},
		{
			name: "first and second valset confirms are valid",
			key:  "/vc/b:" + evmAddress + ":" + signBytes.Hex(),
			values: func() [][]byte {
				signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
				require.NoError(t, err)
				vc1, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress(evmAddress),
					hex.EncodeToString(signature),
				))
				vc2, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress(evmAddress),
					hex.EncodeToString(signature),
				))
				vc3, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			expectedIndex: 0,
			wantErr:       false,
		},
		{
			name: "no valset confirm is valid",
			key:  "/vc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622a:0x1234000000000000000000000000000000000000000000000000000000001234",
			values: func() [][]byte {
				vc1, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				vc2, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				vc3, _ := types.MarshalValsetConfirm(*types.NewValsetConfirm(
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			wantErr: true,
		},
		{
			name:    "empty values slice",
			key:     "/vc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622a:0x1234000000000000000000000000000000000000000000000000000000001234",
			values:  [][]byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualIndex, err := validator.Select(tt.key, tt.values)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIndex, actualIndex)
			}
		})
	}
}

func TestDataCommitmentConfirmValidate(t *testing.T) {
	validator := DataCommitmentConfirmValidator{}

	evmAddress := "0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488"
	privateKey, _ := ethcrypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	nonce := uint64(10)
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), bCommitment)
	signature, err := evm.NewEthereumSignature(dataRootHash.Bytes(), ks, acc)
	require.NoError(t, err)

	tests := []struct {
		name    string
		key     string
		value   []byte
		wantErr bool
	}{
		{
			name: "valid data commitment confirm",
			key:  "/dcc/a:" + evmAddress + ":" + dataRootHash.Hex(),
			value: func() []byte {
				vsc, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					hex.EncodeToString(signature),
					common.HexToAddress(evmAddress),
				))
				return vsc
			}(),
			wantErr: false,
		},
		{
			name:    "invalid key format",
			key:     "/dcc/b/0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value:   nil,
			wantErr: true,
		},
		{
			name:    "invalid key namespace",
			key:     "/dccs/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value:   nil,
			wantErr: true,
		},
		{
			name:    "short key evm address",
			key:     "/dcc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value:   nil,
			wantErr: true,
		},
		{
			name: "not the same evm address in key and in confirm",
			key:  "/dcc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value: func() []byte {
				vsc, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
				))
				return vsc
			}(),
			wantErr: true,
		},
		{
			name: "invalid signature",
			key:  "/dcc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b:0x1234000000000000000000000000000000000000000000000000000000001234",
			value: func() []byte {
				vsc, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"),
				))
				return vsc
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.key, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataCommitmentConfirmSelect(t *testing.T) {
	validator := DataCommitmentConfirmValidator{}

	evmAddress := "0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488"
	privateKey, _ := ethcrypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	nonce := uint64(10)
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), bCommitment)
	signature, err := evm.NewEthereumSignature(dataRootHash.Bytes(), ks, acc)
	require.NoError(t, err)

	tests := []struct {
		name          string
		key           string
		values        [][]byte
		expectedIndex int
		wantErr       bool
	}{
		{
			name: "first data commitment confirm is valid",
			key:  "/dcc/a:" + evmAddress + ":" + dataRootHash.Hex(),
			values: func() [][]byte {
				vc1, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					hex.EncodeToString(signature),
					common.HexToAddress(evmAddress),
				))
				vc2, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress(evmAddress),
				))
				vc3, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			expectedIndex: 0,
			wantErr:       false,
		},
		{
			name: "second data commitment confirm is valid",
			key:  "/dcc/a:" + evmAddress + ":" + dataRootHash.Hex(),
			values: func() [][]byte {
				vc1, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"),
				))
				vc2, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					hex.EncodeToString(signature),
					common.HexToAddress(evmAddress),
				))
				vc3, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			expectedIndex: 1,
			wantErr:       false,
		},
		{
			name: "first and second data commitment confirms are valid",
			key:  "/dcc/a:" + evmAddress + ":" + dataRootHash.Hex(),
			values: func() [][]byte {
				vc1, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					hex.EncodeToString(signature),
					common.HexToAddress(evmAddress),
				))
				vc2, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					hex.EncodeToString(signature),
					common.HexToAddress(evmAddress),
				))
				vc3, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			expectedIndex: 0,
			wantErr:       false,
		},
		{
			name: "no data commitment confirm is valid",
			key:  "/dcc/a:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622a:0x1234000000000000000000000000000000000000000000000000000000001234",
			values: func() [][]byte {
				vc1, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
				))
				vc2, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c"),
				))
				vc3, _ := types.MarshalDataCommitmentConfirm(*types.NewDataCommitmentConfirm(
					"0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
					common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d"),
				))
				return [][]byte{vc1, vc2, vc3}
			}(),
			wantErr: true,
		},
		{
			name:    "empty values slice",
			key:     "/dcc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622a:0x1234000000000000000000000000000000000000000000000000000000001234",
			values:  [][]byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualIndex, err := validator.Select(tt.key, tt.values)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIndex, actualIndex)
			}
		})
	}
}
