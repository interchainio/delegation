package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const UATOM_PER_ATOM = 1000000

// csvData contained in the csv file
type csvData struct {
	Amount  float64 // atoms, up to 6 decimals
	Address string  // cosmos1 acc address
}

func (c csvData) ToMsgSend(from sdk.AccAddress) (msg bank.MsgSend, err error) {
	amount := int64(c.Amount * UATOM_PER_ATOM)
	addr, err := sdk.AccAddressFromBech32(c.Address)
	if err != nil {
		return msg, err
	}
	return bank.MsgSend{
		FromAddress: from,
		ToAddress:   addr,
		Amount:      sdk.Coins{sdk.NewInt64Coin("uatom", amount)},
	}, nil
}

func (c csvData) ToMsgDelegate(from sdk.AccAddress) (msg stakingtypes.MsgDelegate, err error) {
	amount := int64(c.Amount * UATOM_PER_ATOM)
	addr, err := sdk.ValAddressFromBech32(c.Address)
	if err != nil {
		return msg, err
	}
	return stakingtypes.MsgDelegate{
		DelegatorAddress: from,
		ValidatorAddress: addr,
		Amount:           sdk.NewInt64Coin("uatom", amount),
	}, nil
}

type undelegateData struct {
	Amount  sdk.Coin // float64
	Address sdk.ValAddress
}

func ToMsgUndelegate(from sdk.AccAddress, val sdk.ValAddress, coin sdk.Coin) (msg stakingtypes.MsgUndelegate, err error) {
	return stakingtypes.MsgUndelegate{
		DelegatorAddress: from,
		ValidatorAddress: val,
		Amount:           coin,
	}, nil
}

func readTxData(fileName string) ([]csvData, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var data []csvData
	for i, record := range records {
		addr := record[0]
		amount, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("Error on line %d: %v", i, err)
		}
		data = append(data, csvData{amount, addr})
	}
	return data, nil
}
