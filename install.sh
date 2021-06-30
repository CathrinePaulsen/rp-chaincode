#!/bin/bash

# This script is used to automate and simplify the process of updating
# smart contracts on the test network, with consists of two steps:
#   1. Packaging and installing chaincode on all peers              (this script)
#   2. Approving chaincode on all peers                             (see approve.sh)

echo "Packaging/installing chaincode"
read -p 'Enter chaincode name: ' name
read -p 'Enter chaincode version: ' version_number
read -p 'Enter path to chaincode: ' path

# Package smart contract into chaincode
echo "Packaging... This may take a while."
peer lifecycle chaincode package $name.tar.gz --path $path --lang golang --label $name\_$version_number

# Install chaincode at org1 admin
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
peer lifecycle chaincode install $name.tar.gz

# install chaincode at org2 admin
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
peer lifecycle chaincode install $name.tar.gz

# Verify that chaincode has been installed
peer lifecycle chaincode queryinstalled
