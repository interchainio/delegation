# Round 1 - Delegating to GoS Winners

To begin with, the ICF intends to delegate a fraction of its total ATOMs to
certain Game of Stakes winners that are active validators. 
All validators that received recommended  ATOM allocations from Game of Stakes have submitted identifying
information to the ICF and have proven their capabilities in hard won battles during
the Game of Stakes competition. 

To be eligible for delegation, such validators must:

- have been recommended an ATOM allocation in the genesis file as a result of
  Game of Stakes
- have less than 1M ATOMs bonded to their validator at the time the ICF computes
  its intended delegations
- have bonded more than 50% of the ATOMs won during Game of Stakes to their
  validator

The amount delegated will be such that:

- the validator will not have more than 1M bonded to them after the ICF's
  delegation, at the time the ICF computes its intended delegations
- the total amount delegated by the ICF here is otherwise split evenly between validators

## GoS winners using non-GoS validator addresses

Some winners of Game of Stakes (GoS) have noted that they created validators using
addresses from the fundraiser, rather than their Game of Stakes address, and
were thus neglected by the script available here. Since such validators would
otherwise fit the outlined criteria, the ICF may consider delegating to them if
they submit proof of ownership of their GoS and fundraiser addresses by performing the following steps: 

- Open an issue in this repository that includes both the GoS address and the fundraiser address,
  where the fundraiser address corresponds to a currently active validator.
- Link to a transaction on the Cosmos Hub sent from the GoS account that includes the
  fundraiser address in the transaction's memo field.
- Link to a transaction on the Cosmos Hub from the fundraiser address that includes the
  GoS address in the transaction's memo field.

The following instances of this came forward:

| Name | GoS Address | Validator Address | Issue
--------------------------------------------------
| 01no.de | cosmos1wf3sncgk7s2ykamrhy4etf09y94rrrg43cdad7 | cosmosvaloper17mggn4znyeyg25wd7498qxl7r2jhgue8u4qjcq | #10
| stake.zone | cosmos199843dmw5r4nkt6ld00y0wdm0rnnwump4lgs30 | cosmosvaloper1rfpar0qx3umnhu0f6wjp4hvnr3x6u5389e094j | #11
| coinone | cosmos1pz6yu5vdxfzw85cn6d7rp52me4lu8khx7lpkzd | cosmosvaloper1te8nxpc2myjfrhaty0dnzdhs5ahdh5agzuym9v | #13


## Recomputing the Delegation Amounts

The recommended GoS ATOM allocation can be fetched from the [launch
repo](https://github.com/cosmos/launch):

```
curl https://raw.githubusercontent.com/cosmos/launch/master/accounts/icf/gos.json > data/gos.json
```

The resulting delegations from the ICF, according to the above criteria, can be
computed from the `main.go` file in this repository, by specifying the total
amount of ATOM to be delegated. For instance:

```
go run cmd/delegation/main.go 200000
```

Note you must first run `dep ensure` once to fetch the dependencies, and you
must have a locally running and synced `gaiad` node.


## Disclaimer

The ICF reserves the right to change its delegations at any time. It may withdraw delegations
from some or all Game of Stakes winners and/or redelegate to specific validators at any point 
in time without any reason. Further, no validator may assert any claims against the ICF to delegate 
ATOMs to them. The ICF is under no obligation to delegate its ATOMs.

This document is for information purposes only with regards to the ICF's
intention and is not binding in any way. 

