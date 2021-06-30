# rp-chaincode

This repo contains three smart contracts (written in Go) demonstrating the three security vulnerabilities explored in my bachelor thesis: [Revisiting Smart Contract Vulnerabilities in Hyperledger Fabric](https://repository.tudelft.nl/islandora/object/uuid:dd09d153-a9df-4c1b-a317-d93c1231ee28?collection=education). The paper explores three reported smart contract-specific vulnerabilities in Hyperledger Fabric, their methods of exploitation, impact severity estimation and possible countermeasures. The vulnerabilities include global variables, updates using rich queries and pseudorandom number generators.

Each contract include the commands used to deploy and reproduce the exploitation of the contract. The exploitations are explained in more detail in the paper.

**Disclaimer: these contracts are _vulnerable to exploitation_ and should only be used for educational purposes in a designated test environment.**

---
### Setup and Install
It is recommended to use the Hyperledger Fabric test network to deploy these contracts. The specific Hyperledger Fabric version used for the thesis was v2.2.3.

1. Install the prerequisite software (Git, cURL, Docker and Docker Compose) according to [this Hyperledger Fabric guide](https://hyperledger-fabric.readthedocs.io/en/release-2.2/prereqs.html). Additionally, installing [Hyperledger Explorer](https://github.com/hyperledger/blockchain-explorer) is recommended (but not necessary) to more easily inspect the blocks on the blockchain.
2. To install and setup the test network, follow [the official Hyperledger Fabric tutorial](https://hyperledger-fabric.readthedocs.io/en/release-2.2/test_network.html).
3. Make sure that the environment variables are set correctly according to the section "Interacting with the network" in the above tutorial. The `CORE_PEER` environmental variables define the peer used for the `peer` CLI. The `set_peer.sh` script can be used to quickly set the current peer to organization 1 or 2.
---
### Contract deployment
To deploy a contract use the following command:
```
./network.sh deployCC          \
	-ccn <chaincode_name>      \
	-ccp <path_to_chaincode>   \
	-ccl <chancode_language>   \
	-ccep <endorsement_policy>
```
The `-ccep` flag can be left out, in which case the endorsement policy will default to `"AND('Org1MSP.peer','Org2MSP.peer')"`.

---
### Contract invocation
The Hyperledger Fabric `peer` CLI distinguishes between query transactions (read-only) and update transactions.

The following command can be used to invoke **query** transactions with `N` arguments:
```
peer chaincode query  \
  -C <channel_name>   \
  -n <chaincode_name> \
  -c '{"Args":["<function_name>", "<argument1>",...,"<argumentN>"]}'
```
Query transactions are only executed by the peer whose address is stored in `CORE_PEER_ADDRESS`.

Similarly, the following command can be used to invoke **update** transactions with `N` arguments:
```
peer chaincode invoke                     \
  -o <orderer_information>                \
  -C <channel_name> -n <chaincode_name>   \
  --peerAddresses <peer1_information>     \
  --peerAddresses <peer2_information>     \
  -c '{"function":"<function_name>", "Args":["<argument1>,...,<argumentN>]}'
```
This update transaction will be executed by all the peers whose addresses are specified after the `--peerAddresses` flag. You therefore need to include enough peer addresses to pass the endorsement policy.
