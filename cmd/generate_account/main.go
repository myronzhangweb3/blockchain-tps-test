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
	pksPath = "generate_account/pks.txt"
)

func main() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// 从环境变量读取工作账户数量
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

		// 从私钥中获取公钥，然后从公钥中获取地址
		address := crypto.PubkeyToAddress(privateKey.PublicKey)
		// 将私钥转换为字符串形式
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
