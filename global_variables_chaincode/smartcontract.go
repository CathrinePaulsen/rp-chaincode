/*
Copyright 2020 IBM All Rights Reserved.
Copyright 2021 Cathrine Paulsen All Rights Reserved.
SPDX-License-Identifier: Apache-2.0

Based on and inspired by Hyperledger Fabric Samples (https://github.com/hyperledger/fabric-samples/tree/main/asset-transfer-basic/application-go)
*/

/*
=== Deploy ===
./network.sh deployCC \
	-ccn globvar \
	-ccp <path_to_this_directory> \
	-ccl go

=== Initialize ledger ===
peer chaincode invoke \
	-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
	--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
	-C mychannel -n globvar \
	--peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
	--peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
	-c '{"function":"InitLedger","Args":[]}'

=== Query ledger ===
peer chaincode query -C mychannel -n globvar -c '{"Args":["GetAllAssets"]}'

=== Add new asset (ONE PEER) ===
peer chaincode invoke \
	-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
	--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
	-C mychannel -n globvar -c '{"Args":["CreateAsset2","red","0","Alice","0"]}'

=== Add new asset (ALL PEERS) ===
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
 --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
 -C mychannel -n globvar --peerAddresses localhost:7051 \
 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
 --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
 -c '{"function":"CreateAsset2","Args":["red","0","Alice","0"]}'

=== EXPLOITS ===
The smart contract is vulnerable to two exploits: non-determinism and unexpected behavior.
Both of these are explained in the thesis paper, see README.
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Global variables
var totalAssets = 0
var maxAssets = 10

// Asset describes basic details of what makes up a simple asset
type Asset struct {
	ID             string `json:"ID"`
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{ID: strconv.Itoa(1), Color: "blue", Size: 5, Owner: "Tomoko", AppraisedValue: 300},
		{ID: strconv.Itoa(2), Color: "red", Size: 5, Owner: "Brad", AppraisedValue: 400},
	}

	totalAssets = 6

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string) error {
	if totalAssets >= maxAssets {
		return fmt.Errorf("max asset limit has been reached")
	}
	totalAssets++
	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: totalAssets / maxAssets * 100,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	if exists, _ := s.AssetExists(ctx, id); exists {
		return fmt.Errorf("asset already exists with id: %v", id)
	} else {
		return ctx.GetStub().PutState(asset.ID, assetJSON)
	}
}

// ReadAsset returns the asset stored in the world state with given id.
// peer chaincode query -C mychannel -n basic -c '{"Args":["ReadAsset","asset6"]}'
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	// overwriting original asset with new asset
	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// DeleteAsset deletes an given asset from the world state.
func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state.
// peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
// --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
// -C mychannel -n basic --peerAddresses localhost:7051 \
// --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
// --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
// -c '{"function":"TransferAsset","Args":["asset6","Christopher"]}'
// peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem -C mychannel -n basic --peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt -c '{"function":"TransferAsset","Args":["asset6","Christopher"]}'
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newOwner string) error {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}

	asset.Owner = newOwner
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// GetAllAssets returns all assets found in world state
// peer chaincode query -C mychannel -n basic -c '{"Args":["GetAllAssets"]}'
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

func (s *SmartContract) GetTotalAssets() string {
	return strconv.Itoa(totalAssets)
}

func main() {
	assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating global_variables_chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting global_variables_chaincode: %v", err)
	}
}
