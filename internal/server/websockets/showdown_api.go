package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mmf/config"
	"net/http"
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

func checkTransactionOnChain(userConfirmation *UserPayment, userId string) bool {
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

	matchId, ok := params[0].(string)
	if !ok {
		log.Println("Invalid match id")
		return false
	}
	if matchId != userConfirmation.MatchId {
		log.Println("Invalid match id")
		return false
	}

	id, ok := params[1].(string)
	if !ok {
		log.Println("Invalid username")
		return false
	}
	if id != userId {
		log.Println("Invalid username")
		return false
	}

	return true
}

func idToApiKey(userId string) (*ShowdownTokenResponse, error) {
	url := fmt.Sprintf("%s/get_lichess_token?showdownUserID=%s", config.GlobalConfig.ShowdownUserService.URL, userId)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", config.GlobalConfig.ShowdownUserService.ApiKey)

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

	var apiResponse ShowdownApiBulkResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, err
	}

	user, ok := apiResponse[userId]
	if !ok {
		return nil, fmt.Errorf("no lichess token found for user %s", userId)
	}
	return &user, nil
}

func idToWallet(userId string) (*WalletAddressResponse, error) {
	url := fmt.Sprintf("%s/user/info_batch?showdownUserID=%s", config.GlobalConfig.ShowdownApi.URL, userId)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

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

	var apiResponse []WalletAddressResponse
	if err = json.Unmarshal(body, &apiResponse); err != nil {
		return nil, err
	}

	if len(apiResponse) == 0 {
		return nil, fmt.Errorf("no wallet address found for user %s", userId)
	}

	return &apiResponse[0], nil
}

type ShowdownTokenResponse struct {
	LichessId    string `json:"lichessId"`
	LichessToken string `json:"lichessToken"`
}

type WalletAddressResponse struct {
	WalletAddress string `json:"walletAddress"`
}

type ShowdownApiBulkResponse map[string]ShowdownTokenResponse
