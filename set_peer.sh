#!/bin/bash

programname=$0

function usage {
    echo "usage: source $programname [organization_number]"
    echo "  Sets the current peer to the provided organization."
    echo "  The organization number is either 1 or 2. 0 will display the current value that is set."
    echo "  IMPORTANT: the script must be run with source for the export statements to take effect."
    exit 1
}

if [ $# -eq 0 ]; then
    usage
fi

if [ $1 -eq 1 ]; then
    echo "Peer set to Org1"
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID="Org1MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
    export CORE_PEER_ADDRESS=localhost:7051

fi

if [ $1 -eq 2 ]; then
    echo "Peer set to Org2"
    echo $1
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID="Org2MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
    export CORE_PEER_ADDRESS=localhost:9051
fi

if [ $1 -eq 0 ]; then
    echo "I am $CORE_PEER_LOCALMSPID"
fi
