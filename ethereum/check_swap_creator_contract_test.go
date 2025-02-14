// Copyright 2023 The AthanorLabs/atomic-swap Authors
// SPDX-License-Identifier: LGPL-3.0-only

package contracts

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"os"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/athanorlabs/atomic-swap/common"
	"github.com/athanorlabs/atomic-swap/tests"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// deployContract is a test helper that deploys the SwapCreator contract and returns the
// deployed address
func deployContract(
	t *testing.T,
	ec *ethclient.Client,
	pk *ecdsa.PrivateKey,
	trustedForwarder ethcommon.Address,
) ethcommon.Address {
	ctx := context.Background()
	contractAddr, _, err := DeploySwapCreatorWithKey(ctx, ec, pk, trustedForwarder)
	require.NoError(t, err)
	return contractAddr
}

func deployForwarder(t *testing.T, ec *ethclient.Client, pk *ecdsa.PrivateKey) ethcommon.Address {
	addr, err := DeployGSNForwarderWithKey(context.Background(), ec, pk)
	require.NoError(t, err)
	return addr
}

// getContractCode is a test helper that deploys the swap creator contract to read back
// and return the finalised byte code post deployment.
func getContractCode(t *testing.T, trustedForwarder ethcommon.Address) []byte {
	ec, _ := tests.NewEthClient(t)
	pk := tests.GetMakerTestKey(t)
	contractAddr := deployContract(t, ec, pk, trustedForwarder)
	code, err := ec.CodeAt(context.Background(), contractAddr, nil)
	require.NoError(t, err)
	return code
}

func TestCheckForwarderContractCode(t *testing.T) {
	ec, _ := tests.NewEthClient(t)
	pk := tests.GetMakerTestKey(t)
	trustedForwarder := deployForwarder(t, ec, pk)
	err := CheckForwarderContractCode(context.Background(), ec, trustedForwarder)
	require.NoError(t, err)
}

// This test will fail if the compiled SwapCreator contract is updated, but the
// expectedSwapCreatorBytecodeHex constant is not updated. Use this test to update the
// constant.
func TestExpectedSwapCreatorBytecodeHex(t *testing.T) {
	allZeroTrustedForwarder := ethcommon.Address{}
	codeHex := ethcommon.Bytes2Hex(getContractCode(t, allZeroTrustedForwarder))
	require.Equal(t, expectedSwapCreatorBytecodeHex, codeHex,
		"update the expectedSwapCreatorBytecodeHex constant with the actual value to fix this test")
}

// This test will fail if the compiled SwapCreator contract is updated, but the
// forwarderAddrIndexes slice of trusted forwarder locations is not updated. Use
// this test to update the slice.
func TestForwarderAddrIndexes(t *testing.T) {
	ec, _ := tests.NewEthClient(t)
	pk := tests.GetMakerTestKey(t)
	trustedForwarder := deployForwarder(t, ec, pk)
	contactBytes := getContractCode(t, trustedForwarder)

	addressLocations := make([]int, 0) // at the current time, there should always be 2
	for i := 0; i < len(contactBytes)-ethAddrByteLen; i++ {
		if bytes.Equal(contactBytes[i:i+ethAddrByteLen], trustedForwarder[:]) {
			addressLocations = append(addressLocations, i)
			i += ethAddrByteLen - 1 // -1 since the loop will increment by 1
		}
	}

	t.Logf("forwarderAddrIndexes: %v", addressLocations)
	require.EqualValues(t, forwarderAddrIndices, addressLocations,
		"update forwarderAddrIndexes with above logged indexes to fix this test")
}

// Ensure that we correctly verify the SwapCreator contract when initialised with
// different trusted forwarder addresses.
func TestCheckSwapCreatorContractCode(t *testing.T) {
	ec, _ := tests.NewEthClient(t)
	pk := tests.GetMakerTestKey(t)
	trustedForwarderAddrs := []string{
		deployForwarder(t, ec, pk).Hex(),
		deployForwarder(t, ec, pk).Hex(),
		deployForwarder(t, ec, pk).Hex(),
	}

	for _, addrHex := range trustedForwarderAddrs {
		tfAddr := ethcommon.HexToAddress(addrHex)
		contractAddr := deployContract(t, ec, pk, tfAddr)
		parsedTFAddr, err := CheckSwapCreatorContractCode(context.Background(), ec, contractAddr)
		require.NoError(t, err)
		require.Equal(t, addrHex, parsedTFAddr.Hex())
	}
}

// Tests that we fail when the wrong contract byte code is found
func TestCheckSwapCreatorContractCode_fail(t *testing.T) {
	ec, _ := tests.NewEthClient(t)
	pk := tests.GetMakerTestKey(t)

	// Deploy a forwarder contract and then try to verify it as SwapCreator contract
	contractAddr := deployForwarder(t, ec, pk)
	_, err := CheckSwapCreatorContractCode(context.Background(), ec, contractAddr)
	require.ErrorIs(t, err, errInvalidSwapCreatorContract)
}

func TestSepoliaContract(t *testing.T) {
	endpoint := os.Getenv("ETH_SEPOLIA_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://rpc.sepolia.org/"
	}

	// temporarily place a funded sepolia private key below to deploy the test contract
	const sepoliaKey = ""

	ctx := context.Background()
	ec, err := ethclient.Dial(endpoint)
	require.NoError(t, err)
	defer ec.Close()

	parsedTFAddr, err := CheckSwapCreatorContractCode(ctx, ec, common.StagenetConfig().SwapCreatorAddr)
	if errors.Is(err, errInvalidSwapCreatorContract) && sepoliaKey != "" {
		pk, err := ethcrypto.HexToECDSA(sepoliaKey) //nolint:govet // shadow declaration of err
		require.NoError(t, err)
		forwarderAddr := common.StagenetConfig().ForwarderAddr
		sfAddr, _, err := DeploySwapCreatorWithKey(context.Background(), ec, pk, forwarderAddr)
		require.NoError(t, err)
		t.Logf("New Sepolia SwapCreator deployed with TrustedForwarder %s", forwarderAddr)
		t.Fatalf("Update common.StagenetConfig.ContractAddress with %s", sfAddr.Hex())
	} else {
		require.NoError(t, err)
		t.Logf("Sepolia SwapCreator deployed with TrustedForwarder=%s", parsedTFAddr.Hex())
	}
}
