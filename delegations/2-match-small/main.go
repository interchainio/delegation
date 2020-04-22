package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/interchainio/delegation/pkg"
	tmclient "github.com/tendermint/tendermint/rpc/client"
)

var (
	cdc = codec.New()

	// expects a locally running node
	node = tmclient.NewHTTP("https://rpc.cosmos.network:26657", "/websocket")

	// expects the gos.json file to be here
	gosJSON = "data/gos.json"

	// output unsigned delegation tx
	outputFile = "unsigned-delegations.json"

	// gas per msg included in the tx
	gasPerMsg = 210000

	// addresses conflicted out
	conflicts = []string{"cosmos1fghgwhgtxtcshj4a9alp7u2qv6n2wffqj4jdl9"}
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

	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)

	// get list of validators and gos winners,
	validators := pkg.GetValidators(cdc, node)
	gosMap := pkg.ListToMap(gosJSON)

	var (
		minStaked float64 = 1
		maxStaked float64 = 250000
	)
	eligibleVals, ineligibleVals := getSizeEligibleVals(minStaked, maxStaked, validators, conflicts)

	// determine how much to delegate to each validator
	// and collect it as a MsgDelegate
	//N := len(eligibleVals)
	atoms := float64(icfAtoms)
	var msgs []sdk.Msg
	fmt.Println("RANK, ADDRESS, NAME, STAKED, SELF-DELEGATION / GOS-WINNINGS, COMMISSION/MAX-COMMISSION, MAX-COMMISSION-CHANGE - TO-DELEGATE")
	fmt.Println("ELIGIBLE")
	for i, v := range eligibleVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		delegate := math.Min(maxStaked-staked, 0.5*staked)
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
			Amount:            sdk.NewCoin("uatom", sdk.NewInt(int64(delegate*1000000))),
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
	N := 7
	i := 0
	for len(msgs) > 0 {
		n := N
		if len(msgs) < n {
			n = len(msgs)
		}
		pkg.WriteTx(cdc, msgs[:n], gasPerMsg, fmt.Sprintf("%s-%d", outputFile, i))
		msgs = msgs[n:]
		i += 1
	}
}

func getSizeEligibleVals(minStaked, maxStaked float64, validators []stakingtypes.Validator, conflicts []string) ([]stakingtypes.Validator, []stakingtypes.Validator) {
	// determine eligible validators
	var eligibleVals []stakingtypes.Validator
	var ineligibleVals []stakingtypes.Validator

	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Tokens.GT(validators[j].Tokens)
	})

	for _, v := range validators {
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		// eligible if they have:
		// - less than 100k staked
		eligibleAmt := staked > 0 && staked >= minStaked && staked < maxStaked
		conflict := addrInList(v, conflicts)
		if eligibleAmt && !conflict {
			eligibleVals = append(eligibleVals, v)
		} else {
			ineligibleVals = append(ineligibleVals, v)
		}
	}
	return eligibleVals, ineligibleVals
}

func addrInList(v stakingtypes.Validator, conflicts []string) bool {
	for _, c := range conflicts {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		if addr == c {
			return true
		}
	}
	return false
}
