package main

import (
	"blockchain_tps_test/models"
	"blockchain_tps_test/tools"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
	"log"
	"math/big"
	"os"
	"strconv"
)

const (
	pksPath = "pks/pks.txt"
)

func main() {
	_ = godotenv.Load()

	numWorkersStr := os.Getenv("NUM_WORKERS")
	if numWorkersStr == "" {
		log.Fatal("NUM_WORKERS environment variable is required")
	}
	numWorkers, err := strconv.Atoi(numWorkersStr)
	if err != nil {
		log.Fatalf("Invalid NUM_WORKERS value: %v", err)
	}

	workerAccounts, err := generateWorkers(numWorkers)
	if err != nil {
		log.Fatalf("Failed to generate worker accounts: %v", err)
	}

	var pks []string
	for _, account := range workerAccounts {
		pks = append(pks, account.Address.String()+","+account.PrivateKeyHex)
	}
	err = tools.FileWrite(pksPath, pks)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func generateWorkers(numWorkers int) (accounts []models.Account, err error) {
	for range numWorkers {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %v", err)
		}

		address := crypto.PubkeyToAddress(privateKey.PublicKey)
		privateKeyBytes := crypto.FromECDSA(privateKey)
		privateKeyHex := fmt.Sprintf("%x", privateKeyBytes)

		accounts = append(accounts, models.Account{
			Address:       address,
			PrivateKeyHex: privateKeyHex,
			Balance:       big.NewInt(0),
		})
	}

	return accounts, nil
}
