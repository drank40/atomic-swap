// Copyright 2023 The AthanorLabs/atomic-swap Authors
// SPDX-License-Identifier: LGPL-3.0-only

// Package relayer provides libraries for creating and validating relay requests and responses.
package relayer

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	logging "github.com/ipfs/go-log"

	"github.com/athanorlabs/atomic-swap/coins"
	contracts "github.com/athanorlabs/atomic-swap/ethereum"
	"github.com/athanorlabs/atomic-swap/net/message"
)

const (
	relayedClaimGas   = 70000  // worst case gas usage for the claimRelayer swapFactory call
	forwarderClaimGas = 156000 // worst case gas usage when using forwarder to claim
)

// FeeWei and FeeEth are the fixed 0.009 ETH fee for using a swap relayer to claim.
var (
	FeeWei = big.NewInt(9e15)
	FeeEth = coins.NewWeiAmount(FeeWei).AsEther()
)

var log = logging.Logger("relayer")

// CreateRelayClaimRequest fills and returns a RelayClaimRequest ready for
// submission to a relayer.
func CreateRelayClaimRequest(
	ctx context.Context,
	claimerEthKey *ecdsa.PrivateKey,
	ec *ethclient.Client,
	swapCreatorAddr ethcommon.Address,
	forwarderAddr ethcommon.Address,
	swap *contracts.SwapCreatorSwap,
	secret *[32]byte,
) (*message.RelayClaimRequest, error) {

	signature, err := createForwarderSignature(
		ctx,
		claimerEthKey,
		ec,
		swapCreatorAddr,
		forwarderAddr,
		swap,
		secret,
	)
	if err != nil {
		return nil, err
	}

	return &message.RelayClaimRequest{
		OfferID:         nil, // set elsewhere if sending to counterparty
		SwapCreatorAddr: swapCreatorAddr,
		Swap:            swap,
		Secret:          secret[:],
		Signature:       signature,
	}, nil
}
