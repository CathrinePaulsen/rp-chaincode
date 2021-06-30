/*
Copyright 2020 IBM All Rights Reserved.
Copyright 2021 Cathrine Paulsen All Rights Reserved.
SPDX-License-Identifier: Apache-2.0

Based on and inspired by Hyperledger Fabric Samples (https://github.com/hyperledger/fabric-samples/tree/main/asset-transfer-ledger-queries/chaincode-go)
*/

/*
=== Deploy ===
./network.sh deployCC \
	-ccn rich-queries \
	-ccp <path_to_this_directory> \
	-ccl go \
	-ccep "OR('Org1MSP.peer','Org2MSP.peer')"

=== Initialize ledger ===
peer chaincode invoke \
	-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
	--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
	-C mychannel -n couchdb-or \
	--peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
	--peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
	-c '{"function":"InitLedger","Args":[]}'

=== EXPLOIT ===
1. source ./set_peer.sh 1 (compromised peer)    /     source ./set_peer.sh 2
2. Rich query: 		peer chaincode query -C mychannel -n rich-queries -c '{"Args":["QueryAssetsByOwner","Tom"]}'
3. Regular query:	peer chaincode query -C mychannel -n rich-queries -c '{"Args":["ReadAsset","asset1"]}'
4. Modify the appraisedValue of asset1 in the CouchDB instance (localhost:5984/_utils).
5. Change color as the compromised Org1:
	peer chaincode invoke \
		-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
		--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
		-C mychannel -n rich-queries -c '{"Args":["ChangeColorByOwnerFixed","Tom","black"]}'
6. Query both peers to verify that the appraisedValue has been illegally changed on the ledger.

==== Invoke assets ====
Change owner:
	peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
		--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
		-C mychannel -n couchdb -c '{"Args":["TransferAsset","asset1","Tom"]}'
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const index = "color~name"

// SimpleChaincode implements the fabric-contract-api-go programming model
type SimpleChaincode struct {
	contractapi.Contract
}

type Asset struct {
	DocType        string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	ID             string `json:"ID"`      //the field tags are needed to keep case from bouncing around
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}

// HistoryQueryResult structure used for returning result of history query
type HistoryQueryResult struct {
	Record    *Asset    `json:"record"`
	TxId     string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	IsDelete  bool      `json:"isDelete"`
}

// PaginatedQueryResult structure used for returning paginated query results and metadata
type PaginatedQueryResult struct {
	Records             []*Asset `json:"records"`
	FetchedRecordsCount int32    `json:"fetchedRecordsCount"`
	Bookmark            string   `json:"bookmark"`
}

// CreateAsset initializes a new asset in the ledger
func (t *SimpleChaincode) CreateAsset(ctx contractapi.TransactionContextInterface, assetID, color string, size int, owner string, appraisedValue int) error {
	exists, err := t.AssetExists(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to get asset: %v", err)
	}
	if exists {
		return fmt.Errorf("asset already exists: %s", assetID)
	}

	asset := &Asset{
		DocType:        "asset",
		ID:             assetID,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}
	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(assetID, assetBytes)
	if err != nil {
		return err
	}

	//  Create an index to enable color-based range queries, e.g. return all blue assets.
	//  An 'index' is a normal key-value entry in the ledger.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~color~name.
	//  This will enable very efficient state range queries based on composite keys matching indexName~color~*
	colorNameIndexKey, err := ctx.GetStub().CreateCompositeKey(index, []string{asset.Color, asset.ID})
	if err != nil {
		return err
	}
	//  Save index entry to world state. Only the key name is needed, no need to store a duplicate copy of the asset.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	return ctx.GetStub().PutState(colorNameIndexKey, value)
}

// ReadAsset retrieves an asset from the ledger
func (t *SimpleChaincode) ReadAsset(ctx contractapi.TransactionContextInterface, assetID string) (*Asset, error) {
	assetBytes, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset %s: %v", assetID, err)
	}
	if assetBytes == nil {
		return nil, fmt.Errorf("asset %s does not exist", assetID)
	}

	var asset Asset
	err = json.Unmarshal(assetBytes, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// DeleteAsset removes an asset key-value pair from the ledger
func (t *SimpleChaincode) DeleteAsset(ctx contractapi.TransactionContextInterface, assetID string) error {
	asset, err := t.ReadAsset(ctx, assetID)
	if err != nil {
		return err
	}

	err = ctx.GetStub().DelState(assetID)
	if err != nil {
		return fmt.Errorf("failed to delete asset %s: %v", assetID, err)
	}

	colorNameIndexKey, err := ctx.GetStub().CreateCompositeKey(index, []string{asset.Color, asset.ID})
	if err != nil {
		return err
	}

	// Delete index entry
	return ctx.GetStub().DelState(colorNameIndexKey)
}

// TransferAsset transfers an asset by setting a new owner name on the asset
func (t *SimpleChaincode) TransferAsset(ctx contractapi.TransactionContextInterface, assetID, newOwner string) error {
	asset, err := t.ReadAsset(ctx, assetID)
	if err != nil {
		return err
	}

	asset.Owner = newOwner
	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(assetID, assetBytes)
}

// constructQueryResponseFromIterator constructs a slice of assets from the resultsIterator
func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) ([]*Asset, error) {
	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var asset Asset
		err = json.Unmarshal(queryResult.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetAssetsByRange performs a range query based on the start and end keys provided.
// Read-only function results are not typically submitted to ordering. If the read-only
// results are submitted to ordering, or if the query is used in an update transaction
// and submitted to ordering, then the committing peers will re-execute to guarantee that
// result sets are stable between endorsement time and commit time. The transaction is
// invalidated by the committing peers if the result set has changed between endorsement
// time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
func (t *SimpleChaincode) GetAssetsByRange(ctx contractapi.TransactionContextInterface, startKey, endKey string) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

// TransferAssetByColor will transfer assets of a given color to a certain new owner.
// Uses GetStateByPartialCompositeKey (range query) against color~name 'index'.
// Committing peers will re-execute range queries to guarantee that result sets are stable
// between endorsement time and commit time. The transaction is invalidated by the
// committing peers if the result set has changed between endorsement time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
// Example: GetStateByPartialCompositeKey/RangeQuery
func (t *SimpleChaincode) TransferAssetByColor(ctx contractapi.TransactionContextInterface, color, newOwner string) error {
	// Execute a key range query on all keys starting with 'color'
	coloredAssetResultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(index, []string{color})
	if err != nil {
		return err
	}
	defer coloredAssetResultsIterator.Close()

	for coloredAssetResultsIterator.HasNext() {
		responseRange, err := coloredAssetResultsIterator.Next()
		if err != nil {
			return err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return err
		}

		if len(compositeKeyParts) > 1 {
			returnedAssetID := compositeKeyParts[1]
			asset, err := t.ReadAsset(ctx, returnedAssetID)
			if err != nil {
				return err
			}
			asset.Owner = newOwner
			assetBytes, err := json.Marshal(asset)
			if err != nil {
				return err
			}
			err = ctx.GetStub().PutState(returnedAssetID, assetBytes)
			if err != nil {
				return fmt.Errorf("transfer failed for asset %s: %v", returnedAssetID, err)
			}
		}
	}

	return nil
}

// QueryAssetsByOwner queries for assets based on the owners name.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// Example: Parameterized rich query
func (t *SimpleChaincode) QueryAssetsByOwner(ctx contractapi.TransactionContextInterface, owner string) ([]*Asset, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)
	return getQueryResultForQueryString(ctx, queryString)
}

func (t *SimpleChaincode) ChangeColorByOwnerVulnerable(ctx contractapi.TransactionContextInterface, owner string, color string) error {
	queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)
	byteArray, err := getQueryResultForQueryString(ctx, queryString)
	if err != nil {
		return err
	}

	for _, asset := range byteArray {
		asset.Color = color
		assetBytes, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *SimpleChaincode) ChangeColorByOwnerFixed(ctx contractapi.TransactionContextInterface, owner string, color string) error {
	queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)
	byteArray, err := getQueryResultForQueryString(ctx, queryString)
	if err != nil {
		return err
	}

	for _, asset := range byteArray {
		key := asset.ID
		assetBytes, _ := ctx.GetStub().GetState(key) // safe key-based query
		_ = json.Unmarshal(assetBytes, asset)
		asset.Color = color
		assetBytes, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// QueryAssets uses a query string to perform a query for assets.
// Query string matching state database syntax is passed in and executed as is.
// Supports ad hoc queries that can be defined at runtime by the client.
// If this is not desired, follow the QueryAssetsForOwner example for parameterized queries.
// Only available on state databases that support rich query (e.g. CouchDB)
// Example: Ad hoc rich query
func (t *SimpleChaincode) QueryAssets(ctx contractapi.TransactionContextInterface, queryString string) ([]*Asset, error) {
	return getQueryResultForQueryString(ctx, queryString)
}

// getQueryResultForQueryString executes the passed in query string.
// The result set is built and returned as a byte array containing the JSON results.
func getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

// GetAssetsByRangeWithPagination performs a range query based on the start and end key,
// page size and a bookmark.
// The number of fetched records will be equal to or lesser than the page size.
// Paginated range queries are only valid for read only transactions.
// Example: Pagination with Range Query
func (t *SimpleChaincode) GetAssetsByRangeWithPagination(ctx contractapi.TransactionContextInterface, startKey string, endKey string, pageSize int, bookmark string) ([]*Asset, error) {

	resultsIterator, _, err := ctx.GetStub().GetStateByRangeWithPagination(startKey, endKey, int32(pageSize), bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

// QueryAssetsWithPagination uses a query string, page size and a bookmark to perform a query
// for assets. Query string matching state database syntax is passed in and executed as is.
// The number of fetched records would be equal to or lesser than the specified page size.
// Supports ad hoc queries that can be defined at runtime by the client.
// If this is not desired, follow the QueryAssetsForOwner example for parameterized queries.
// Only available on state databases that support rich query (e.g. CouchDB)
// Paginated queries are only valid for read only transactions.
// Example: Pagination with Ad hoc Rich Query
func (t *SimpleChaincode) QueryAssetsWithPagination(ctx contractapi.TransactionContextInterface, queryString string, pageSize int, bookmark string) (*PaginatedQueryResult, error) {

	return getQueryResultForQueryStringWithPagination(ctx, queryString, int32(pageSize), bookmark)
}

// getQueryResultForQueryStringWithPagination executes the passed in query string with
// pagination info. The result set is built and returned as a byte array containing the JSON results.
func getQueryResultForQueryStringWithPagination(ctx contractapi.TransactionContextInterface, queryString string, pageSize int32, bookmark string) (*PaginatedQueryResult, error) {

	resultsIterator, responseMetadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	assets, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	return &PaginatedQueryResult{
		Records:             assets,
		FetchedRecordsCount: responseMetadata.FetchedRecordsCount,
		Bookmark:            responseMetadata.Bookmark,
	}, nil
}

// GetAssetHistory returns the chain of custody for an asset since issuance.
func (t *SimpleChaincode) GetAssetHistory(ctx contractapi.TransactionContextInterface, assetID string) ([]HistoryQueryResult, error) {
	log.Printf("GetAssetHistory: ID %v", assetID)

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(assetID)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &asset)
			if err != nil {
				return nil, err
			}
		} else {
			asset = Asset{
				ID: assetID,
			}
		}

		timestamp, err := ptypes.Timestamp(response.Timestamp)
		if err != nil {
			return nil, err
		}

		record := HistoryQueryResult{
			TxId:      response.TxId,
			Timestamp: timestamp,
			Record:    &asset,
			IsDelete:  response.IsDelete,
		}
		records = append(records, record)
	}

	return records, nil
}

// AssetExists returns true when asset with given ID exists in the ledger.
func (t *SimpleChaincode) AssetExists(ctx contractapi.TransactionContextInterface, assetID string) (bool, error) {
	assetBytes, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return false, fmt.Errorf("failed to read asset %s from world state. %v", assetID, err)
	}

	return assetBytes != nil, nil
}

// InitLedger creates the initial set of assets in the ledger.
func (t *SimpleChaincode) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{DocType: "asset", ID: "asset1", Color: "blue", Size: 5, Owner: "Tom", AppraisedValue: 300},
		{DocType: "asset", ID: "asset2", Color: "red", Size: 5, Owner: "Brad", AppraisedValue: 400},
		{DocType: "asset", ID: "asset3", Color: "green", Size: 10, Owner: "Jin Soo", AppraisedValue: 500},
		{DocType: "asset", ID: "asset4", Color: "yellow", Size: 10, Owner: "Max", AppraisedValue: 600},
		{DocType: "asset", ID: "asset5", Color: "black", Size: 15, Owner: "Adriana", AppraisedValue: 700},
		{DocType: "asset", ID: "asset6", Color: "white", Size: 15, Owner: "Michel", AppraisedValue: 800},
	}

	for _, asset := range assets {
		err := t.CreateAsset(ctx, asset.ID, asset.Color, asset.Size, asset.Owner, asset.AppraisedValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SimpleChaincode{})
	if err != nil {
		log.Panicf("Error creating asset chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting asset chaincode: %v", err)
	}
}
