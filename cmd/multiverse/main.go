package main

import (
	"fmt"
	"os"

	gaia "github.com/cosmos/cosmos-sdk/cmd/gaia/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/interchainio/delegation/pkg"

	"github.com/spf13/cobra"
)

var (
	cdc = gaia.MakeCodec()

	flagInputFileName  string
	flagOutputFileName string
	flagSenderAddr     string

	// gas per msg included in the tx
	gasPerMsg = 200000
)

const icfAddr = "cosmos1unc788q8md2jymsns24eyhua58palg5kc7cstv"

func init() {
	RootCmd.PersistentFlags().StringVarP(&flagInputFileName, "csv", "", "msgs.csv", "csv file containing addresses and amounts")
	RootCmd.PersistentFlags().StringVarP(&flagOutputFileName, "output", "", "unsigned-delegations.json", "location to output json file")
	RootCmd.PersistentFlags().StringVarP(&flagSenderAddr, "sender", "", icfAddr, "sender address")

	RootCmd.AddCommand(
		TransferCmd,
		DelegateCmd,
	)
}

var RootCmd = &cobra.Command{
	Use:   "multiverse",
	Short: "A tool to convert csv files to txs",
	Long:  "A tool to convert csv files to txs",
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

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
	pkg.WriteTxs(cdc, txData, gasPerMsg, flagOutputFileName, pkg.NanoSMsgsPerTx)
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
	pkg.WriteTxs(cdc, txData, gasPerMsg, flagOutputFileName, pkg.NanoSMsgsPerTx)
	return nil
}
