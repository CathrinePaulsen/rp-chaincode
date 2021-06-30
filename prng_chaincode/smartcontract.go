/*
Copyright 2021 Cathrine Paulsen All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

/*
INITIALIZE LEDGER:
peer chaincode invoke -o localhost:7050 \
	--ordererTLSHostnameOverride orderer.example.com \
	--tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
	-C mychannel -n vulnlottery --peerAddresses localhost:7051 \
	--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
	--peerAddresses localhost:9051 \
	--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
	-c '{"function":"InitLedger","Args":[]}'

SUBMIT GUESS:
peer chaincode invoke -o localhost:7050 \
	--ordererTLSHostnameOverride orderer.example.com \
	--tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
	-C mychannel -n vulnlottery --peerAddresses localhost:7051 \
	--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
	--peerAddresses localhost:9051 \
	--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
	-c '{"function":"GuessNumber","Args":["guess"]}'

DEPLOY:
./network.sh deployCC -ccn vulnlottery -ccp <path_to_this_directory> -ccl go

QUERY:
peer chaincode query -C mychannel -n vulnlottery -c '{"Args":["GetCurrentWin"]}'

=== EXPLOIT ===
After initializing the ledger with the current winning number, use e.g. Hyperledger Explorer to inspect the blockchain.
The timestamp used to generate the winning number is available, and can be used to calculate the correct winning
number.
*/


package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"math/rand"
	"strconv"
)

type SmartContract struct {
	contractapi.Contract
}

type Ticket struct {
	Owner	string 	`json:"owner"`
	Guess	string	`json:"guess"`
	Won     bool    `json:"won"`
}

type Win struct {
	ID              string  `json:"id"`
	Number			string	`json:"number"`
	Won 			bool	`json:"won"`
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	err := s.generateNewWin(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *SmartContract) GuessNumber(ctx contractapi.TransactionContextInterface, guess string) error {
	currentWin, err := s.GetCurrentWin(ctx)
	if err != nil {
		return err
	}

	var ticket Ticket
	owner, _ := ctx.GetClientIdentity().GetID()

	if currentWin.Number == guess {
		ticket = Ticket{
			Owner:	owner,
			Guess:	guess,
			Won: 	true,
		}

		err = s.generateNewWin(ctx)
		if err != nil {
			return err
		}
	} else {
		ticket = Ticket{
			Owner:	owner,
			Guess:	guess,
			Won: 	false,
		}
	}

	assetJSON, err := json.Marshal(ticket)
	if err != nil {
		return err
	}


	return ctx.GetStub().PutState(ticket.Owner, assetJSON)
}

func (s *SmartContract) generateNewWin(ctx contractapi.TransactionContextInterface) error {
	// Generate random number
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	rand.Seed(timestamp.GetSeconds())
	randomNumber := rand.Int()

	h := sha256.New()
	h.Write([]byte(strconv.Itoa(randomNumber)))
	hashedNumber := hex.EncodeToString(h.Sum(nil))

	// Create new winning number
	asset := Win{
		ID: "current_win",
		Number: hashedNumber,
		Won: false,
	}

	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	// Store the winning number
	err = ctx.GetStub().PutState(asset.ID, assetBytes)
	if err != nil {
		return err
	}

	return nil
}

// GetCurrentWin function is public for demo purposes
func (s *SmartContract) GetCurrentWin(ctx contractapi.TransactionContextInterface) (*Win, error) {
	id := "current_win"
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Win
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// GetRandom Test function
func (s *SmartContract) GetRandom() string {
	return strconv.Itoa(rand.Int())
}

// GetRandomTime Test function
func (s *SmartContract) GetRandomTime(ctx contractapi.TransactionContextInterface) string {
	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	rand.Seed(timestamp.GetSeconds())
	return strconv.Itoa(rand.Int())
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating asset chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting asset chaincode: %v", err)
	}
}