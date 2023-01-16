package types_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestValidateMsgDataCommitmentConfirm(t *testing.T) {
	var (
		evmAddress                   = common.HexToAddress("0xb462864E395d88d6bc7C5dd5F3F5eb4cc2599255")
		cosmosAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, 20)
	)
	specs := map[string]struct {
		beginBlock uint64
		endBlock   uint64
		expErr     bool
	}{
		"all good": {
			beginBlock: 1,
			endBlock:   200,
			expErr:     false,
		},
		"begin block higher than end block": {
			beginBlock: 10,
			endBlock:   5,
			expErr:     true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			println(fmt.Sprintf("Spec is %v", msg))
			msg := types.NewMsgDataCommitmentConfirm(
				"commitment",
				"signature",
				cosmosAddress,
				evmAddress,
				spec.beginBlock,
				spec.endBlock,
				20,
			)
			// when
			err := msg.ValidateBasic()
			if spec.expErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
