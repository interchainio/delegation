package main

import (
	"fmt"
	"sort"
	"strconv"

	gaia "github.com/cosmos/cosmos-sdk/cmd/gaia/app"
	"github.com/interchainio/delegation/pkg"
	tmclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/spf13/cobra"
)

var (
	cdc = gaia.MakeCodec()

	fullNodeURL string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&fullNodeUrl, "url", "", "localhost:26657", "URL of synced full-node to use.")
}

var RootCmd = &cobra.Command{
	Use:   "stake-dist",
	Short: "Display stake distribution at latest block height",
	Long: `Display how many validators control 1/3 and 2/3 of
	the total stake, at the latest block height.`,
	Run: getDist,
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		panic(err)
	}
}

func getDist(cmd *cobra.Command, args []string) {
	node := tmclient.NewHTTP(fullNodeURL, "/websocket")

	var toAdd float64
	if len(args) == 1 {
		toAddInt, err := strconv.Atoi(args[0])
		if err != nil {
			panic(err)
		}
		toAdd += float64(toAddInt)
	}

	// get list of validators and sort descending by power
	validators := pkg.GetValidators(cdc, node)
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Tokens.GT(validators[j].Tokens)
	})

	var totalStaked float64 = 0
	for _, v := range validators {
		staked := pkg.UatomIntToAtomFloat(v.Tokens)
		totalStaked += staked
	}
	totalStaked += toAdd

	var accum float64 = 0
	oneThird := 0
	twoThirds := 0
	for i, v := range validators {
		staked := pkg.UatomIntToAtomFloat(v.Tokens)
		accum += staked
		if twoThirds == 0 && accum > 0.666666666666*totalStaked {
			twoThirds = i + 1
		}
		if oneThird == 0 && accum > 0.333333333333*totalStaked {
			oneThird = i + 1
		}
	}
	fmt.Printf("At block height %d\n", pkg.GetLatestHeight(node))
	fmt.Printf("%d validators control 2/3 of the stake\n", twoThirds)
	fmt.Printf("%d validators control 1/3 of the stake\n", oneThird)
}
