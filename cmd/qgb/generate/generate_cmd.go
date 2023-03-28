package generate

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	util "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
)

// Command helper command to generate a new Ed25519 private key in hex format.
// This will be used to generate the key needed when starting the orchestrator or relayer.
// Will be removed once we support a key management tool.
func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generates a new Ed25519 private key in hex format",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("generating a new Ed25519 private key in hex format...")

			sr := util.NewTimeSeededRand()
			priv, _, err := crypto.GenerateEd25519Key(sr)
			if err != nil {
				fmt.Println(err.Error())
				return err
			}

			bytez, err := priv.Raw()
			if err != nil {
				fmt.Println(err.Error())
				return err
			}
			fmt.Println(hexutil.Encode(bytez)[2:])
			return nil
		},
	}
}
