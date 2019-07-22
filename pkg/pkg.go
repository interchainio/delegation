package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	amino "github.com/tendermint/go-amino"
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

	// to query all an accounts delegations
	delegationsEndPath = "subspace"
	delegationsPath    = fmt.Sprintf("/store/%s/%s", storeName, delegationsEndPath)
)

// fetch validators from /abci_query
func GetValidators(cdc *amino.Codec, node *tmclient.HTTP) staking.Validators {
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

func GetDelegations(cdc *amino.Codec, node *tmclient.HTTP, addr []byte) staking.Delegations {
	key := staking.GetDelegationsKey(sdk.AccAddress(addr))
	opts := tmclient.ABCIQueryOptions{
		Prove: false,
	}

	resQuery, err := node.ABCIQueryWithOptions(delegationsPath, key, opts)
	if err != nil {
		panic(err)
	}
	resRaw := resQuery.Response.Value

	if len(resRaw) == 0 {
		return nil
	}

	var resKVs []sdk.KVPair
	cdc.MustUnmarshalBinaryLengthPrefixed(resRaw, &resKVs)

	var delegations staking.Delegations
	for _, kv := range resKVs {
		delegations = append(delegations, types.MustUnmarshalDelegation(cdc, kv.Value))
	}
	return delegations

}

// fetch self delegations from /abci_query
func GetSelfDelegation(cdc *amino.Codec, node *tmclient.HTTP, addr []byte) float64 {
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

	return UatomIntToAtomFloat(delegation.Shares.TruncateInt())
}

// fetch latest block height from /status
func GetLatestHeight(node *tmclient.HTTP) int64 {
	resQuery, err := node.Status()
	if err != nil {
		panic(err)
	}
	return resQuery.SyncInfo.LatestBlockHeight
}

func DecToFloat(d sdk.Dec) float64 {
	d100 := d.Mul(sdk.NewDec(100))
	return float64(d100.TruncateInt64()) / 100
}

// convert Int uatoms to float64 atoms
func UatomIntToAtomFloat(i sdk.Int) float64 {
	return float64(i.Int64()) / float64(1000000)
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
		amounts[addr] += amt
	}
	return amounts
}

// LedgerNanoS can sign about this many msgs at once
const NanoSMsgsPerTx = 7

// WriteTxs splits the msgs into multiple txs according to msgsPerTx, each named like "{fileNamePrefix}-{index}"
func WriteTxs(cdc *amino.Codec, msgs []sdk.Msg, gasPerMsg int, fileNamePrefix string, msgsPerTx int) {
	// split it up
	N := msgsPerTx
	i := 0
	for len(msgs) > 0 {
		n := N
		if len(msgs) < n {
			n = len(msgs)
		}
		WriteTx(cdc, msgs[:n], gasPerMsg, fmt.Sprintf("%s-%d", fileNamePrefix, i))
		msgs = msgs[n:]
		i += 1
	}
}

// WriteTx writes the set of msgs to the file
func WriteTx(cdc *amino.Codec, msgs []sdk.Msg, gasPerMsg int, fileName string) {
	tx := auth.StdTx{
		Msgs: msgs,
		Fee: auth.StdFee{
			Gas: uint64(gasPerMsg * len(msgs)),
		},
	}
	bz, err := cdc.MarshalJSONIndent(tx, "", "  ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fileName, bz, 0600)
	if err != nil {
		panic(err)
	}
}
