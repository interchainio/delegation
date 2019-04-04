package commands

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"

	gaia "github.com/cosmos/cosmos-sdk/cmd/gaia/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/interchainio/delegation/pkg"
	tmclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/spf13/cobra"
)

var (
	cdc = gaia.MakeCodec()

	fullNodeUrl string
	gosJSON     string
	outputFile  string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&fullNodeUrl, "url", "", "localhost:26657", "URL of synced full-node to use.")
	RootCmd.PersistentFlags().StringVarP(&gosJSON, "gos-json", "", "data/gos.json", "source of json file")
	RootCmd.PersistentFlags().StringVarP(&outputFile, "output", "", "unsigned-delegations.json", "location to output json file")
}

var RootCmd = &cobra.Command{
	Use:   "delegation",
	Short: "the delegation command",
	Long:  "for something",
	Run:   getDelegation,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		panic(err)
	}
}

func getDelegation(cmd *cobra.Command, args []string) {
	node := tmclient.NewHTTP(fullNodeUrl, "/websocket")
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
	var eligibleVals []stakingtypes.Validator
	var ineligibleVals []stakingtypes.Validator

	var million float64 = 1000000
	for _, v := range gosVals {
		addr := sdk.AccAddress(v.OperatorAddress).String()
		gosAmt := gosMap[addr]
		selfDelegation := pkg.GetSelfDelegation(cdc, node, v.OperatorAddress)
		staked := pkg.UatomIntToAtomFloat(v.Tokens)

		// eligible if they have:
		// - less than 1M staked
		// - self bonded more than half their gos winnings
		eligibleAmt := selfDelegation*2 > gosAmt && staked < million
		if eligibleAmt {
			eligibleVals = append(eligibleVals, v)
		} else {
			ineligibleVals = append(ineligibleVals, v)
		}
	}

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
		delegate := math.Min(million-staked, propAmt)
		atoms -= delegate

		maxRate := decToFloat(v.Commission.MaxRate)
		commission, commissionChange := decToFloat(v.Commission.Rate), decToFloat(v.Commission.MaxChangeRate)

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

		maxRate := decToFloat(v.Commission.MaxRate)
		commission, commissionChange := decToFloat(v.Commission.Rate), decToFloat(v.Commission.MaxChangeRate)

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
		writeTx(msgs[:n], fmt.Sprintf("%s-%d", outputFile, i))
		msgs = msgs[n:]
		i += 1
	}
}

func writeTx(msgs []sdk.Msg, fileName string) {
	tx := auth.StdTx{
		Msgs: msgs,
		Fee: auth.StdFee{
			Gas: uint64(200000 * len(msgs)),
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

func decToFloat(d sdk.Dec) float64 {
	d100 := d.Mul(sdk.NewDec(100))
	return float64(d100.TruncateInt64()) / 100
}
