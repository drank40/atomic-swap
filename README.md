# ETH-XMR Atomic Swaps

This is a prototype of ETH<->XMR atomic swaps, which was worked on during ETHLisbon.

### Protocol

Alice has ETH and wants XMR, Bob has XMR and wants ETH. They come to an agreement to do the swap and the amounts they will swap.

#### Ethereum smart contract interface

Offline phase:
- Both parties Alice and Bob select a secret: `s_a` & `s_b`, which are used to construct valid points on the ed25519 curve: `P_ed_a` and `P_ed_b` accordingly. The parties share their public keys with each other.

##### Step 1.
Alice creates a smart contract on Ethereum and sends her ether to it. The contract has the following properties:
- it is non-destructible
- it is initiated with `P_ed_a` & `P_ed_b`
- it has a `ready` function, which can only be called by Alice. Once `ready` is invoked (in step 3), Bob can proceed with redeeming his ether. Alice has `t_0` time period to call `ready`, otherwise all funds are transferred to Bob.
- it has a `refund` function that can only be called by Alice, and only before `ready` is called. Once `ready` is invoked, Alice can no longer call `refund` within the next time duration`t_1`. After `t_1` has passed though, and Bob hasn't called `redeem`, `refund` is re-enabled to prevent a dead-lock.
- `refund` takes one parameter from Alice: `s_a`. This allows Alice to get her ether back in case Bob mis-behaves, but as it's public from now on, Bob can still redeem his monero (in case he was offline). 
- `redeem` takes one parameter from Bob: `s_b`. At this point, Alice found out Bob's secret and can claim Monero by combining her and Bob's secrets.
- both the `refund` & `redeem` functions check, whether the argument `s_x` provided indeed corresponds to the public key `P_ed_x` and only if true, allows fund transfer.

##### Step 2. 
Bob sees the smart contract being created. He sends his monero to an account address constructed from `P_ed_a + P_ed_b`.

The funds can only be accessed by an entity having both `s_a` & `s_b`.

##### Step 3.
Alice sees that monero has been sent. She calls `ready` on the smart contract.
From this point on, Bob can redeem his ether by calling `redeem(s_b)`.

By redeeming, Bob revealed his secret. Now Alice is the only one that has both `s_a` & `s_b` and she can access the monero in the account created from `P_ed_a + P_ed_b`.

#### What could go wrong

##### Step 2.

- Alice locked her ether, but Bob doesn't lock his monero.
Alice never calls `ready`, and can instead call `refund(s_a)` within `t_0`, getting her ether back.


##### Step 3.

- Alice called `ready`, but Bob never redeems. Deadlocks are prevented thanks to a second timelock `t_1`, which re-enables Alice to call refund.

- Alice never calls `ready` within `t_1`. Bob can still claim his ETH, after `t_1` has passed, and XMR remains locked forever.


### Instructions

Start ganache-cli with determinstic keys:
```
ganache-cli -d
```

Start monerod for regtest:
```
./monerod --regtest --fixed-difficulty=1 --rpc-bind-port 18081 --offline
```

Start monero-wallet-rpc for Bob with some wallet that has regtest monero:
```
./monero-wallet-rpc  --rpc-bind-port 18083 --password "" --disable-rpc-login --wallet-file test-wallet
```

Start monero-wallet-rpc for Alice:
```
./monero-wallet-rpc  --rpc-bind-port 18084 --password "" --disable-rpc-login --wallet-dir .
```

Build binary:
```
./scripts/build.sh
```

This creates an `atomic-swap` binary in the root directory.

To run as Alice, execute:
```
./atomic-swap --amount 1 --alice
```

Alice will print out a libp2p node address, for example `/ip4/127.0.0.1/tcp/9933/p2p/12D3KooWBW1cqB9t5fKP8yZPq3PcWcgbvuNai5ZpAeWFAbs5RNAA`. This will be used for Bob to connect.

To run as Bob and connect to Alice, replace the bootnode in the following line with what Alice logged, and execute:

```
./atomic-swap --amount 1 --bob --bootnodes /ip4/127.0.0.1/tcp/9933/p2p/12D3KooWBW1cqB9t5fKP8yZPq3PcWcgbvuNai5ZpAeWFAbs5RNAA
```

If all goes well, you should see Alice and Bob successfully exchange messages and execute the swap protocol. The result is that Alice now owns the private key to a Monero account (and is the only owner of that key) and Bob has the ETH transferred to him.


##### Compiling contract bindings

Download solc v0.8.9

Set `SOLC_BIN` to the downloaded binary
```
export SOLC_BIN=solc
```

Generate the bindings
```
./scripts/generate-bindings.sh
```

##### Testing
To run tests on the go bindings, execute
```
go test ./swap-contract
```
