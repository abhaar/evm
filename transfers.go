package main

import (
	"context"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

var tracer = map[string]string{
	"tracer": "callTracer",
}

func ParseInternalTransfers(ctx context.Context, url string, txHash common.Hash) ([]Transfer, error) {
	client, err := rpc.Dial(url)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	var trace Call
	err = client.CallContext(ctx, &trace, "debug_traceTransaction", txHash.Hex(), tracer)
	if err != nil {
		log.Fatal(err)
	}

	transfers, err := parse(trace)
	if err != nil {
		return nil, err
	}
	return transfers, nil
}

func parse(call Call) ([]Transfer, error) {
	var transfers []Transfer
	// Ignore calls that have an error and reverted
	if call.Error != "" {
		return nil, nil
	} else {
		// Recursively parse child calls
		for _, childCall := range call.Calls {
			childTransfers, err := parse(childCall)
			if err != nil {
				return nil, err
			}

			transfers = append(transfers, childTransfers...)
		}

		// add the current call
		transfer, err := getTransfer(call)
		if err != nil {
			return nil, err
		}

		transfers = append(transfers, transfer)
	}

	return transfers, nil
}

func getTransfer(call Call) (Transfer, error) {
	// Only CALL types contain value transfers
	if call.Type != "CALL" {
		return Transfer{}, nil
	}

	value, err := hexutil.DecodeBig(call.Value)
	if err != nil {
		return Transfer{}, err
	}

	// Ignore 0 value transfers
	if value.Cmp(big.NewInt(0)) <= 0 {
		return Transfer{}, nil
	}

	return Transfer{
		From:  common.HexToAddress(call.From),
		To:    common.HexToAddress(call.To),
		Value: *value,
	}, nil
}

type Call struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Calls []Call `json:"calls"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Error string `json:"error"`
}

// type call struct {
// 	From    string `json:"from"`
// 	Gas     string `json:"gas"`
// 	GasUsed string `json:"gasUsed"`
// 	To      string `json:"to"`
// 	Input   string `json:"input"`
// 	Output  string `json:"output"`
// 	Calls   []call `json:"calls"`
// 	Value   string `json:"value"`
// 	Type    string `json:"type"`
// 	Error   string `json:"error"`
// }

type Transfer struct {
	From  common.Address
	To    common.Address
	Value big.Int
}
