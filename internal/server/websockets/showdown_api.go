package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mmf/config"
	"mmf/internal/model"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const abiJSON = `[{
	"inputs": [
		{
			"internalType": "string",
			"name": "_matchId",
			"type": "string"
		},
		{
			"internalType": "string",
			"name": "_chessUsername",
			"type": "string"
		},
		{
			"internalType": "uint256",
			"name": "_amount",
			"type": "uint256"
		}
	],
	"name": "joinMatch",
	"outputs": [],
	"stateMutability": "nonpayable",
	"type": "function"
}]`

func checkTransactionOnChain(userConfirmation *UserPayment, ticket *model.MemberData) bool {
	txHash := common.HexToHash(userConfirmation.TxnHash)

	client, err := ethclient.Dial(config.GlobalConfig.EthRpc.URL)
	if err != nil {
		log.Println("Error connecting to eth client: ", err)
		return false
	}

	tx, _, err := client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		log.Println("Error getting transaction by hash: ", err)
		return false
	}

	if tx == nil {
		return false
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		log.Println("Error parsing abi: ", err)
		return false
	}

	data := tx.Data()

	method, err := parsedABI.MethodById(data[:4])
	if err != nil {
		log.Println("Error getting method by id: ", err)
		return false
	}

	params, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		log.Println("Error unpacking data: ", err)
		return false
	}

	if len(params) != 3 {
		log.Println("Invalid number of parameters")
		return false
	}

	// if params[0].(string) != userConfirmation.MatchId {
	// 	log.Println("Invalid match id")
	// 	return false
	// }

	id, ok := params[1].(string)
	if !ok {
		log.Println("Invalid username")
		return false
	}

	if id != ticket.Id {
		log.Println("Invalid username")
		return false
	}

	return true
}

func usernameToKey(username string) (*string, error) {
	showdownApi := os.Getenv("SHOWDOWN_API")
	showdownKey := os.Getenv("SHOWDOWN_API_KEY")

	url := fmt.Sprintf("%s/get_lichess_token?userID=%s", showdownApi, username)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", showdownKey)

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var apiResponse ShowdownApiResponse
	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		return nil, err
	}

	return &apiResponse.Token, nil
}
