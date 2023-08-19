package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/stretchr/testify/require"

	celestiatestnode "github.com/celestiaorg/celestia-app/test/util/testnode"
	"github.com/cosmos/cosmos-sdk/codec"
	abci "github.com/tendermint/tendermint/abci/types"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

// NodeEVMPrivateKey the key used to initialize the test node validator.
// Its corresponding address is: "0x9c2B12b5a07FC6D719Ed7646e5041A7E85758329".
var NodeEVMPrivateKey, _ = crypto.HexToECDSA("64a1d6f0e760a8d62b4afdde4096f16f51b401eaaecc915740f71770ea76a8ad")

// CelestiaNetwork is a Celestia-app validator running in-process.
// The EVM key that was used to create this network's single validator can
// be retrieved using: `celestiatestnode.NodeEVMPrivateKey`
type CelestiaNetwork struct {
	celestiatestnode.Context
	Accounts []string
	RPCAddr  string
	GRPCAddr string
}

// NewCelestiaNetwork creates a new CelestiaNetwork.
// Uses `testing.T` to fail if an error happens.
// Only supports the creation of a single validator currently.
func NewCelestiaNetwork(ctx context.Context, t *testing.T, genesisOpts ...celestiatestnode.GenesisOption) *CelestiaNetwork {
	if testing.Short() {
		// The main reason for skipping these tests in short mode is to avoid detecting unrelated
		// race conditions.
		// In fact, this test suite uses an existing Celestia-app node to simulate a real environment
		// to execute tests against. However, this leads to data races in multiple areas.
		// Thus, we can skip them as the races detected are not related to this repo.
		t.Skip("skipping tests in short mode.")
	}

	// we create an arbitrary number of funded accounts
	accounts := make([]string, 300)
	for i := 0; i < 300; i++ {
		accounts[i] = tmrand.Str(9)
	}

	tmCfg := celestiatestnode.DefaultTendermintConfig()
	tmCfg.Consensus.TimeoutCommit = time.Millisecond * 5
	appConf := celestiatestnode.DefaultAppConfig()

	clientContext, _, _ := celestiatestnode.NewNetwork(
		t,
		celestiatestnode.DefaultConfig().
			WithAppConfig(appConf).
			WithTendermintConfig(tmCfg).
			WithAccounts(accounts).
			WithGenesisOptions(genesisOpts...).
			WithChainID("qgb-test"),
	)

	appRPC := clientContext.GRPCClient.Target()
	status, err := clientContext.Client.Status(ctx)
	require.NoError(t, err)

	// register EVM address
	rec, err := clientContext.Keyring.Key("validator")
	require.NoError(t, err)
	pubKey, err := rec.GetPubKey()
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromHex(pubKey.Address().String())
	require.NoError(t, err)
	RegisterEVMAddress(
		t,
		clientContext,
		valAddr,
		gethcommon.HexToAddress("0x9c2B12b5a07FC6D719Ed7646e5041A7E85758329"),
	)

	return &CelestiaNetwork{
		Context:  clientContext,
		Accounts: accounts,
		GRPCAddr: appRPC,
		RPCAddr:  status.NodeInfo.ListenAddr,
	}
}

// SetDataCommitmentWindowParams will set the provided data commitment window as genesis state.
func SetDataCommitmentWindowParams(codec codec.Codec, params types.Params) celestiatestnode.GenesisOption {
	return func(state map[string]json.RawMessage) map[string]json.RawMessage {
		qgbGenState := types.DefaultGenesis()
		qgbGenState.Params = &params
		state[types.ModuleName] = codec.MustMarshalJSON(qgbGenState)
		return state
	}
}

// SetDataCommitmentWindow will use the validator account to set the data commitment
// window param. It assumes that the governance params have been set to
// allow for fast acceptance of proposals, and will fail the test if the
// parameters are not set as expected.
func (cn *CelestiaNetwork) SetDataCommitmentWindow(t *testing.T, window uint64) {
	account := "validator"

	// create and submit a new param change proposal for the data commitment window
	change := proposal.NewParamChange(
		types.ModuleName,
		string(types.ParamsStoreKeyDataCommitmentWindow),
		fmt.Sprintf("\"%d\"", window),
	)
	content := proposal.NewParameterChangeProposal(
		"data commitment window update",
		"description",
		[]proposal.ParamChange{change},
	)
	addr := getAddress(account, cn.Context.Keyring)

	msg, err := v1beta1.NewMsgSubmitProposal(
		content,
		sdk.NewCoins(
			sdk.NewCoin(app.BondDenom, sdk.NewInt(1000000000000))),
		addr,
	)
	require.NoError(t, err)

	ecfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	res, err := celestiatestnode.SignAndBroadcastTx(ecfg, cn.Context.Context, account, msg)
	require.Equal(t, res.Code, abci.CodeTypeOK, res.RawLog)
	require.NoError(t, err)
	resp, err := cn.Context.WaitForTx(res.TxHash, 10)
	require.NoError(t, err)
	require.Equal(t, abci.CodeTypeOK, resp.TxResult.Code)

	require.NoError(t, cn.Context.WaitForNextBlock())

	// query the proposal to get the id
	gqc := v1.NewQueryClient(cn.Context.GRPCClient)
	gresp, err := gqc.Proposals(
		cn.Context.GoContext(),
		&v1.QueryProposalsRequest{
			ProposalStatus: v1.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD,
		},
	)
	require.NoError(t, err)
	require.Len(t, gresp.Proposals, 1)

	// create and submit a new vote
	vote := v1.NewMsgVote(
		getAddress(account, cn.Context.Keyring),
		gresp.Proposals[0].Id,
		v1.VoteOption_VOTE_OPTION_YES,
		"",
	)
	res, err = celestiatestnode.SignAndBroadcastTx(ecfg, cn.Context.Context, account, vote)
	require.NoError(t, err)
	resp, err = cn.Context.WaitForTx(res.TxHash, 10)
	require.NoError(t, err)
	require.Equal(t, abci.CodeTypeOK, resp.TxResult.Code)

	// wait for the voting period to complete
	time.Sleep(time.Second * 5)

	// check that the parameters got updated as expected
	bqc := types.NewQueryClient(cn.Context.GRPCClient)
	presp, err := bqc.Params(cn.Context.GoContext(), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, window, presp.Params.DataCommitmentWindow)
}

func RegisterEVMAddress(
	t *testing.T,
	input celestiatestnode.Context,
	valAddr sdk.ValAddress,
	evmAddr gethcommon.Address,
) {
	registerMsg := types.NewMsgRegisterEVMAddress(valAddr, evmAddr)
	res, err := celestiatestnode.SignAndBroadcastTx(
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
		input.Context,
		"validator",
		registerMsg,
	)
	resp, err := input.WaitForTx(res.TxHash, 10)
	require.NoError(t, err)
	require.Equal(t, abci.CodeTypeOK, resp.TxResult.Code)

	require.NoError(t, input.WaitForNextBlock())
}

func getAddress(account string, kr keyring.Keyring) sdk.AccAddress {
	rec, err := kr.Key(account)
	if err != nil {
		panic(err)
	}
	addr, err := rec.GetAddress()
	if err != nil {
		panic(err)
	}
	return addr
}
