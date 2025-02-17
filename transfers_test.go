package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var debugTrace = []byte(`{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "from": "0xe78d5a85c8dbb345683b213be22484d0cdf51065",
        "gas": "0x16dde",
        "gasUsed": "0x162cd",
        "to": "0x6b156d8388dede287ee17689da0cc8eeeda1fcbc",
        "input": "0xbfa20351000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000a81482ac1089a80b0b9d6d803b88f67f7ab5fd35000000000000000000000000b750edf608a2774ec8dbc06961e8664ea4a0a2e5",
        "calls": [
            {
                "from": "0x6b156d8388dede287ee17689da0cc8eeeda1fcbc",
                "gas": "0x8fc",
                "gasUsed": "0x0",
                "to": "0xa81482ac1089a80b0b9d6d803b88f67f7ab5fd35",
                "input": "0x",
                "value": "0x5",
                "type": "CALL"
            },
            {
                "from": "0x6b156d8388dede287ee17689da0cc8eeeda1fcbc",
                "gas": "0x8fc",
                "gasUsed": "0x0",
                "to": "0xb750edf608a2774ec8dbc06961e8664ea4a0a2e5",
                "input": "0x",
                "value": "0x5",
                "type": "CALL"
            }
        ],
        "value": "0xa",
        "type": "CALL"
    }
}`)

func Test_ParseInternalTransfers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newTestServer(t)
	defer server.Close()

	endpoint, _ := url.Parse(server.URL)

	transfers, err := ParseInternalTransfers(ctx, endpoint.String(), common.HexToHash("0x9d6a4fa9215cbb95d92aa56a599b91c6f4cd76234954cd8e01bc83d43d578977"))

	assert.NoError(t, err)
	assert.Equal(t, 3, len(transfers))

	transferMap := make(map[common.Address][]Transfer)
	for _, transfer := range transfers {
		if _, ok := transferMap[transfer.From]; !ok {
			transferMap[transfer.From] = make([]Transfer, 0)
		}
		transferMap[transfer.From] = append(transferMap[transfer.From], transfer)
	}

	aliceAddress := common.HexToAddress("0xe78d5a85c8dbb345683b213be22484d0cdf51065")
	contractAddress := common.HexToAddress("0x6b156d8388dede287ee17689da0cc8eeeda1fcbc")
	bobAddress := common.HexToAddress("0xa81482ac1089a80b0b9d6d803b88f67f7ab5fd35")
	charlieAddress := common.HexToAddress("0xb750edf608a2774ec8dbc06961e8664ea4a0a2e5")

	transferFromAlice := transferMap[aliceAddress]
	transfersFromContract := transferMap[contractAddress]

	assert.Equal(t, 1, len(transferFromAlice))
	assert.Equal(t, contractAddress, transferFromAlice[0].To)
	// Sent 10 Wei to smart-contract
	assert.Equal(t, 10, int(transferFromAlice[0].Value.Int64()))

	assert.Equal(t, 2, len(transfersFromContract))
	assert.Equal(t, bobAddress, transfersFromContract[0].To)
	assert.Equal(t, charlieAddress, transfersFromContract[1].To)

	// Sent 10 Wei from smart-contract to Bob and Charlie
	assert.Equal(t, 10, int(transfersFromContract[0].Value.Int64())+int(transfersFromContract[1].Value.Int64()))
}

func newTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(debugTrace)
				if err != nil {
					t.Fail()
				}
			},
		),
	)
}
