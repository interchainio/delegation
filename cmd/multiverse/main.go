package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/interchainio/delegation/pkg"
	tmclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/spf13/cobra"
)

var (
	cdc = codec.New()

	flagInputFileName  string
	flagOutputFileName string
	flagSenderAddr     string
	flagFullNodeURL    string

	// gas per msg included in the tx
	gasPerMsg = 200000
)

const icfAddr = "cosmos1unc788q8md2jymsns24eyhua58palg5kc7cstv"

func init() {
	RootCmd.PersistentFlags().StringVarP(&flagInputFileName, "csv", "", "msgs.csv", "csv file containing addresses and amounts")
	RootCmd.PersistentFlags().StringVarP(&flagOutputFileName, "output", "", "unsigned-delegations.json", "location to output json file")
	RootCmd.PersistentFlags().StringVarP(&flagSenderAddr, "sender", "", icfAddr, "sender address")

	UndelegateCmd.PersistentFlags().StringVarP(&flagFullNodeURL, "node", "", "localhost:26657", "address of a full node")

	RootCmd.AddCommand(
		AddressCmd,
		TransferCmd,
		DelegateCmd,
		UndelegateCmd,
	)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
}

var RootCmd = &cobra.Command{
	Use:   "multiverse",
	Short: "A tool to convert csv files to txs",
	Long:  "A tool to convert csv files to txs",
}

var AddressCmd = &cobra.Command{
	Use:   "addr",
	Short: "Convert between address formats",
	Long:  "Convert between address formats",
	RunE:  cmdAddress,
}

var TransferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Convert csv to MsgSend",
	Long:  "Convert csv to MsgSend",
	RunE:  cmdTransfer,
}

var DelegateCmd = &cobra.Command{
	Use:   "delegate",
	Short: "Convert csv to MsgDelegate",
	Long:  "Convert csv to MsgDelegate",
	RunE:  cmdDelegate,
}

var UndelegateCmd = &cobra.Command{
	Use:   "undelegate",
	Short: "Undelegate from all inactive validators",
	Long:  "Undelegate from all inactive validators",
	RunE:  cmdUndelegate,
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cmdAddress(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Requires an address in any format")
	}

	addrString := args[0]

	accAddr, err := sdk.AccAddressFromBech32(addrString)
	if err == nil {
		printAddrs(accAddr)
		return nil
	}

	valAddr, err := sdk.ValAddressFromBech32(addrString)
	if err == nil {
		printAddrs(valAddr)
		return nil
	}

	hexAddr, err := sdk.AccAddressFromHex(addrString)
	if err == nil {
		printAddrs(hexAddr)
		return nil
	}
	return fmt.Errorf("Invalid addr")
}

func printAddrs(addr []byte) {
	fmt.Printf("%s\n", sdk.AccAddress(addr))
	fmt.Printf("%s\n", sdk.ValAddress(addr))
}

func cmdTransfer(cmd *cobra.Command, args []string) error {
	from, err := sdk.AccAddressFromBech32(flagSenderAddr)
	if err != nil {
		return fmt.Errorf("Invalid sender: %v", err)
	}

	records, err := readTxData(flagInputFileName)
	if err != nil {
		return err
	}

	var txData []sdk.Msg
	for i, r := range records {
		msg, err := r.ToMsgSend(from)
		if err != nil {
			return fmt.Errorf("Error in line %d: %v", i, err)
		}
		txData = append(txData, msg)
	}
	pkg.WriteTxs(cdc, txData, gasPerMsg, flagOutputFileName, pkg.MsgsPerTxSend)
	return nil
}

func cmdDelegate(cmd *cobra.Command, args []string) error {
	from, err := sdk.AccAddressFromBech32(flagSenderAddr)
	if err != nil {
		return fmt.Errorf("Invalid sender: %v", err)
	}

	records, err := readTxData(flagInputFileName)
	if err != nil {
		return err
	}

	var txData []sdk.Msg
	for i, r := range records {
		msg, err := r.ToMsgDelegate(from)
		if err != nil {
			return fmt.Errorf("Error in line %d: %v", i, err)
		}
		txData = append(txData, msg)
	}
	pkg.WriteTxs(cdc, txData, gasPerMsg, flagOutputFileName, pkg.MsgsPerTxDelegation)
	return nil
}

func cmdUndelegate(cmd *cobra.Command, args []string) error {
	from, err := sdk.AccAddressFromBech32(flagSenderAddr)
	if err != nil {
		return fmt.Errorf("Invalid sender: %v", err)
	}

	node := tmclient.NewHTTP(flagFullNodeURL, "/websocket")

	vals := pkg.GetValidators(cdc, node)

	dels := pkg.GetDelegations(cdc, node, from)
	absentVals := []undelegateData{}
	for _, del := range dels {
		valAddr := del.ValidatorAddress
		if in, v := valInVals(valAddr, vals); !in {
			amt := del.Shares // val.Tokens.ToDec().Mul(del.Shares).Quo(val.DelegatorShares).TruncateInt().Int64()
			coins := amt.Mul(v.Tokens.ToDec()).Quo(v.DelegatorShares).TruncateInt().Int64()
			fmt.Println(coins)
			fmt.Printf("%s, %d\n", valAddr, amt)
			absentVals = append(absentVals, undelegateData{
				Amount:  sdk.NewInt64Coin("uatom", coins), // float64(amt),
				Address: valAddr,
			})
		}
	}

	var txData []sdk.Msg
	for i, absentVal := range absentVals {
		msg, err := ToMsgUndelegate(from, absentVal.Address, absentVal.Amount)
		if err != nil {
			return fmt.Errorf("Error in line %d: %v", i, err)
		}
		txData = append(txData, msg)
	}
	pkg.WriteTxs(cdc, txData, gasPerMsg, flagOutputFileName, pkg.MsgsPerTxDelegation)
	return nil
}

func valInVals(val sdk.ValAddress, vals staking.Validators) (bool, *staking.Validator) {
	for _, v := range vals {
		if bytes.Equal(val, v.OperatorAddress) {
			if v.Status == sdk.Bonded {
				return true, &v
			} else {
				return false, &v
			}
		}
	}
	return false, nil

}
