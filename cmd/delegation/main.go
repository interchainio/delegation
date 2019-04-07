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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/interchainio/delegation/pkg"
	tmclient "github.com/tendermint/tendermint/rpc/client"
)

var (
	cdc = gaia.MakeCodec()

	// expects a locally running node
	node = tmclient.NewHTTP("localhost:26657", "/websocket")

	// expects the gos.json file to be here
	gosJSON = "data/gos.json"

	// output unsigned delegation tx
	outputFile = "unsigned-delegations.json"

	// gas per msg included in the tx
	gasPerMsg = 200000
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Please specify total amount of atoms to delegate")
		os.Exit(1)
	}

	// if a second arg is specified, output the delegation tx
	var delegatorAddr sdk.AccAddress
	if len(args) == 2 {
		var err error
		delegatorAddr, err = sdk.AccAddressFromBech32(args[1])
		if err != nil {
			panic(err)
		}
	}

	icfAtoms, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		panic(err)
	}

	// get list of validators and gos winners,
	validators := pkg.GetValidators(cdc, node)
	gosMap := pkg.ListToMap(gosJSON)

	var maxStaked float64 = 1000000
	eligibleVals, ineligibleVals = getGoSEligibleVals(maxStaked, gosMap, validators)

	// determine how much to delegate to each validator
	// and collect it as a MsgDelegate
	N := len(eligibleVals)
	atoms := float64(icfAtoms)
	var msgs []sdk.Msg
	fmt.Println("RANK, ADDRESS, NAME, STAKED, SELF-DELEGATION / GOS-WINNINGS, COMMISSION/MAX-COMMISSION, MAX-COMMISSION-CHANGE - TO-DELEGATE")
	fmt.Println("ELIGIBLE")
	for i, v := range eligibleVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		propAmt := float64(atoms) / float64(N-i)
		delegate := math.Min(maxStaked-staked, propAmt)
		atoms -= delegate

		maxRate := pkg.DecToFloat(v.Commission.MaxRate)
		commission, commissionChange := pkg.DecToFloat(v.Commission.Rate), pkg.DecToFloat(v.Commission.MaxChangeRate)

		fmt.Printf("%d, %s, %s, %.2f, %d/%d, %.2f/%.2f, %.2f - %.2f\n",
			i, addr, v.Description.Moniker,
			staked, int64(selfDelegation), int64(gosAmt),
			commission, maxRate, commissionChange,
			delegate)

		msgs = append(msgs, stakingtypes.MsgDelegate{
			DelegatorAddress: delegatorAddr,
			ValidatorAddress: v.OperatorAddress,
			Value:            sdk.NewCoin("uatom", sdk.NewInt(int64(delegate*1000000))),
		})
	}

	fmt.Println("INELIGIBLE")
	for i, v := range ineligibleVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		maxRate := pkg.DecToFloat(v.Commission.MaxRate)
		commission, commissionChange := pkg.DecToFloat(v.Commission.Rate), pkg.DecToFloat(v.Commission.MaxChangeRate)

		fmt.Printf("%d, %s, %s, %.2f, %d/%d, %.2f/%.2f, %.2f\n",
			i, addr, v.Description.Moniker,
			staked, int64(selfDelegation), int64(gosAmt),
			commission, maxRate, commissionChange)
	}

	fmt.Println("ATOMs left:", atoms)

	if len(delegatorAddr) == 0 {
		return
	}

	// split it up
	N = 7
	i := 0
	for len(msgs) > 0 {
		n := N
		if len(msgs) < n {
			n = len(msgs)
		}
		pkg.WriteTx(msgs[:n], gasPerMsg, fmt.Sprintf("%s-%d", outputFile, i))
		msgs = msgs[n:]
		i += 1
	}
}

func getGoSEligibleVals(maxStaked float64, gosMap map[string]float64, validators []stakingtypes.Validator) ([]stakingtypes.Validator, []stakingtypes.Validator) {
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
	var eligibleVals []stakingtypes.Validator
	var ineligibleVals []stakingtypes.Validator

	for _, v := range gosVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		// eligible if they have:
		// - less than 1M staked
		// - self bonded more than half their gos winnings
		eligibleAmt := selfDelegation*2 > gosAmt && staked < maxStaked
		if eligibleAmt {
			eligibleVals = append(eligibleVals, v)
		} else {
			ineligibleVals = append(ineligibleVals, v)
		}
	}
	return eligibleVals, ineligibleVals
}
