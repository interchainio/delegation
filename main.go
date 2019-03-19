package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"

	gaia "github.com/cosmos/cosmos-sdk/cmd/gaia/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	tmclient "github.com/tendermint/tendermint/rpc/client"
)

// for querying staking info
var (
	storeName = "staking"

	// to query the list of validators
	validatorsEndPath = "subspace"
	validatorsKey     = []byte{0x21} // staking.ValidatorsKey
	validatorsPath    = fmt.Sprintf("/store/%s/%s", storeName, validatorsEndPath)

	// to query a validators self delegation
	delegationEndPath = "key"
	delegationPath    = fmt.Sprintf("/store/%s/%s", storeName, delegationEndPath)
)

var (
	cdc = gaia.MakeCodec()

	// expects a locally running node
	node = tmclient.NewHTTP("localhost:26657", "/websocket")
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
	validators := getValidators()
	gosMap := ListToMap("gos.json")

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
		selfDelegation := getSelfDelegation(v.OperatorAddress)
		staked := uatomIntToAtomFloat(v.Tokens)

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
		selfDelegation := getSelfDelegation(v.OperatorAddress)
		staked := uatomIntToAtomFloat(v.Tokens)

		propAmt := float64(atoms) / float64(N-i)
		delegate := math.Min(million-staked, propAmt)
		atoms -= delegate

		fmt.Printf("%d, %s, %s, %.2f, %d/%d, %.2f\n", i, addr, v.Description.Moniker, staked, int64(selfDelegation), int64(gosAmt), delegate)
	}
	fmt.Println(atoms)
}

// convert Int uatoms to float64 atoms
func uatomIntToAtomFloat(i sdk.Int) float64 {
	return float64(i.Int64()) / float64(1000000)
}

// fetch validators from /abci_query
func getValidators() staking.Validators {
	opts := tmclient.ABCIQueryOptions{
		Prove: false,
	}

	resQuery, err := node.ABCIQueryWithOptions(validatorsPath, validatorsKey, opts)
	if err != nil {
		panic(err)
	}
	resRaw := resQuery.Response.Value

	var resKVs []sdk.KVPair
	cdc.MustUnmarshalBinaryLengthPrefixed(resRaw, &resKVs)

	var validators staking.Validators
	for _, kv := range resKVs {
		validators = append(validators, types.MustUnmarshalValidator(cdc, kv.Value))
	}
	return validators
}

// fetch self delegations from /abci_query
func getSelfDelegation(addr []byte) float64 {
	key := staking.GetDelegationKey(sdk.AccAddress(addr), sdk.ValAddress(addr))
	opts := tmclient.ABCIQueryOptions{
		Prove: false,
	}

	resQuery, err := node.ABCIQueryWithOptions(delegationPath, key, opts)
	if err != nil {
		panic(err)
	}
	resRaw := resQuery.Response.Value

	if len(resRaw) == 0 {
		return 0
	}
	delegation, err := types.UnmarshalDelegation(cdc, resRaw)
	if err != nil {
		panic(err)
	}

	return uatomIntToAtomFloat(delegation.Shares.TruncateInt())
}

// Load a flattened list of (addr, amt) pairs into a map
// and consolidate any duplicates.
// Panics on odd length, prints duplicates.
func ListToMap(file string) map[string]float64 {
	bz, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	var l []interface{}
	err = json.Unmarshal(bz, &l)
	if err != nil {
		panic(err)
	}

	// list should be pairs of addr, amt
	if len(l)%2 != 0 {
		panic(fmt.Errorf("list length is odd"))
	}

	// loop through two at a time and add the amt to the entry
	// in the map for the addr
	amounts := make(map[string]float64)
	for i := 0; i < len(l); i += 2 {
		addr := l[i].(string)
		amt := l[i+1].(float64)
		if _, ok := amounts[addr]; ok {
			// fmt.Println("Duplicate addr, consolidating", addr)
		}
		amounts[addr] += amt
	}
	return amounts
}
