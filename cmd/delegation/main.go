package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	gaia "github.com/cosmos/cosmos-sdk/cmd/gaia/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/interchainio/delegation/pkg"
	tmclient "github.com/tendermint/tendermint/rpc/client"
)

var (
	cdc = gaia.MakeCodec()

	// expects a locally running node
	node = tmclient.NewHTTP("localhost:26657", "/websocket")

	// expects the gos.json file to be here
	gosJSON = "data/gos.json"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Please specify total amount of atoms to delegate")
		os.Exit(1)
	}
	icfAtoms, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		panic(err)
	}

	// get list of validators and list of gos winners,
	// cross reference them, and sort by voting power
	validators := pkg.GetValidators(cdc, node)
	gosMap := pkg.ListToMap(gosJSON)

	var gosVals []staking.Validator
	for _, v := range validators {
		vs := sdk.AccAddress(v.OperatorAddress).String()
		_, ok := gosMap[vs]
		if ok {
			gosVals = append(gosVals, v)
		}
	}

	sort.Slice(gosVals, func(i, j int) bool {
		return gosVals[i].Tokens.GT(gosVals[j].Tokens)
	})

	// determine eligible validators
	var eligibleVals []staking.Validator

	i := 0
	var million float64 = 1000000
	fmt.Println("RANK, ADDRESS, NAME, STAKED, SELF-DELEGATION / GOS-WINNINGS")
	for _, v := range gosVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		// eligible if they have less than 1M staked and they self bonded more than half their gos winnings:
		eligible := selfDelegation*2 > gosAmt && staked < million
		if eligible {
			eligibleVals = append(eligibleVals, v)
			i++
		}
	}

	// determien how much to delegate to each validator
	N := len(eligibleVals)
	atoms := float64(icfAtoms)
	for i, v := range eligibleVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		propAmt := float64(atoms) / float64(N-i)
		delegate := math.Min(million-staked, propAmt)
		atoms -= delegate

		fmt.Printf("%d, %s, %s, %.2f, %d/%d, %.2f\n", i, addr, v.Description.Moniker, staked, int64(selfDelegation), int64(gosAmt), delegate)
	}
	fmt.Println("ATOMs left:", atoms)
}
