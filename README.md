# ICF ATOM Delegation

This repository summarizes the Interchain Foundation's approach to delegating
its ATOMs.

## Background

The Interchain Foundation (ICF) takes a conservative approach to all of its activity.
As outlined in a [recent blog
post](https://blog.cosmos.network/open-decentralized-networks-87e6097536a3), 
the success of the network depends
fundamentally on the actions of the wider community. This means the community of ATOM holders
is responsible for ensuring the network is sufficiently decentralized and that
community interests are adequately represented by the validator set.

That said, as a significant stake holder in the Cosmos Network, the ICF is
interested, like other significant stake holders, in the success of the network,
which includes the decentralization of its validator set.

To this extent, the ICF intends to delegate some of its ATOMs at some times in a way that
encourages a healthy validator set.

Currently, 4 validators control one third of the stake and 10 control two
thirds. ATOM holders are *strongly encouraged* to diversify their delegations to
improve the decentralization of the validator set.

Also, to achieve decentralization on governance procedures, ICF will stay neutral and 
not overwrite any governance proposal votes done by the validators whom ICF delegates to.

## Delegating to GoS Winners

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
- not be composed of any personnel who is/was a member of Cosmos/Tendermint team

The amount delegated will be such that:

- the validator will not have more than 1M bonded to them after the ICF's
  delegation, at the time the ICF computes its intended delegations
- the total amount delegated by the ICF here is otherwise split evenly between validators

## Disclaimer

The ICF reserves the right to change its delegations at any time. It may withdraw delegations
from some or all Game of Stakes winners and/or redelegate to specific validators at any point 
in time without any reason. Further, no validator may assert any claims against the ICF to delegate 
ATOMs to them. The ICF is under no obligation to delegate its ATOMs.

This document is for information purposes only with regards to the ICF's
intention and is not binding in any way. 

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
