package main

import (
	"blockchain_tps_test/models"
	"blockchain_tps_test/tools"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/joho/godotenv"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	pksPath = "generate_account/pks.txt"
)

var ethRpcUrl, mainAccountPrivateKey string
var numTxsPerWorker uint64
var gasPriceRate uint64
var repayGasFee, minGasFee *big.Int

func main() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// 从环境变量读取配置
	ethRpcUrl = os.Getenv("ETH_RPC_URL")
	if ethRpcUrl == "" {
		log.Fatal("ETH_RPC_URL environment variable is required")
	}

	mainAccountPrivateKey = os.Getenv("MAIN_ACCOUNT_PRIVATE_KEY")
	if mainAccountPrivateKey == "" {
		log.Fatal("MAIN_ACCOUNT_PRIVATE_KEY environment variable is required")
	}

	numTxsPerWorkerStr := os.Getenv("NUM_TXS_PER_WORKER")
	if numTxsPerWorkerStr == "" {
		log.Fatal("NUM_TXS_PER_WORKER environment variable is required")
	}
	numTxsPerWorker, err = strconv.ParseUint(numTxsPerWorkerStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid NUM_TXS_PER_WORKER value: %v", err)
	}

	gasPriceRateStr := os.Getenv("GAS_PRICE_RATE")
	if gasPriceRateStr == "" {
		gasPriceRate = 100 // 默认值
	} else {
		gasPriceRate, err = strconv.ParseUint(gasPriceRateStr, 10, 64)
		if err != nil {
			log.Fatalf("Invalid GAS_PRICE_RATE value: %v", err)
		}
	}

	repayGasFeeStr := os.Getenv("REPAY_GAS_FEE")
	if repayGasFeeStr == "" {
		repayGasFee = big.NewInt(10000000000000000) // 默认 0.01 ETH
	} else {
		repayGasFee = new(big.Int)
		repayGasFee.SetString(repayGasFeeStr, 10)
	}

	minGasFeeStr := os.Getenv("MIN_BALANCE")
	if minGasFeeStr == "" {
		minGasFee = big.NewInt(1000000000000000) // 默认 0.001 ETH
	} else {
		minGasFee = new(big.Int)
		minGasFee.SetString(minGasFeeStr, 10)
	}

	client, err := ethclient.Dial(ethRpcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// 签署交易。这需要链ID，对于主网是1。
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("error getting chain ID: %s", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(int64(gasPriceRate)))
	gasPrice = gasPrice.Div(gasPrice, big.NewInt(100))

	// 加载主账户
	mainAccountKey, err := crypto.HexToECDSA(mainAccountPrivateKey)
	if err != nil {
		log.Fatalf("Invalid main workerAccount private key: %v", err)
	}
	mainAccountAddress := crypto.PubkeyToAddress(mainAccountKey.PublicKey)
	log.Printf("mainAccountAddress: %s\n", mainAccountAddress)

	// 加载工作账户
	workerAccounts, err := loadWorkers(pksPath)
	if err != nil {
		log.Fatalf("Failed to generate worker accounts: %v", err)
	}

	for {
		fmt.Println("---------------- next loop ----------------\b")
		// 发送ETH
		prepayGasFee2(chainId, workerAccounts, client, gasPrice, mainAccountAddress.String())

		// 构建模拟交易
		rawTxList := buildTestTx(chainId, workerAccounts, client, mainAccountAddress, gasPrice)

		// 广播交易
		broadcastTransactions(client.Client(), rawTxList)

		time.Sleep(10 * time.Second)
	}
}

func prepayGasFee(chainID *big.Int, workerAccounts []models.Account, client *ethclient.Client, gasPrice *big.Int, mainAccountAddress string) {
	mainAccountNonce, err := client.NonceAt(context.Background(), common.HexToAddress(mainAccountAddress), nil)
	if err != nil {
		log.Fatalf("Failed to get mainAccountNonce: %v", err)
	}

	wg := sync.WaitGroup{}
	for i, workerAccount := range workerAccounts {
		wg.Add(1)
		go func(i int, nonce *big.Int, workerAccount string) {
			defer wg.Done()
			// 如果余额足够不再发送
			balance, err := client.BalanceAt(context.Background(), common.HexToAddress(workerAccount), nil)
			if balance.Cmp(minGasFee) >= 0 {
				return
			}
			// 从主账户向工作账户发送ETH
			_, _, err = buildEthTx(ethRpcUrl, mainAccountPrivateKey, chainID, nonce, workerAccount, repayGasFee, big.NewInt(21000), gasPrice, true, false, nil)
			if err != nil {
				log.Printf("Failed to send ETH to worker %d: %v. from: %s, nonce: %d", i, err, workerAccount, nonce)
			}
			log.Printf("Sent ETH to worker %d\n", i)
		}(i, big.NewInt(int64(mainAccountNonce+uint64(i))), workerAccount.Address.String())
	}
	wg.Wait()
}

func prepayGasFee2(chainID *big.Int, workerAccounts []models.Account, client *ethclient.Client, gasPrice *big.Int, mainAccountAddress string) {
	log.Printf("---------------- prepayGasFee2 ----------------\b")

	wg := sync.WaitGroup{}
	mainAccountNonce, err := client.NonceAt(context.Background(), common.HexToAddress(mainAccountAddress), nil)
	if err != nil {
		log.Fatalf("Failed to get mainAccountNonce: %v", err)
	}

	nonce := big.NewInt(int64(mainAccountNonce))

	for i, workerAccount := range workerAccounts {
		if i%100 == 0 {
			fmt.Println("checking worker", i)
		}
		// 如果余额足够不再发送
		balance, err := client.BalanceAt(context.Background(), workerAccount.Address, nil)
		if balance.Cmp(minGasFee) >= 0 {
			continue
		}
		// 从主账户向工作账户发送ETH
		wg.Add(1)
		_, _, err = buildEthTx(ethRpcUrl, mainAccountPrivateKey, chainID, nonce, workerAccount.Address.String(), repayGasFee, big.NewInt(21000), gasPrice, true, true, &wg)
		if err != nil {
			log.Printf("Failed to send ETH to worker %d: %v. from: %s, nonce: %d", i, err, workerAccount, nonce)
			wg.Done()
		}
		nonce = nonce.Add(nonce, big.NewInt(1))
		log.Printf("Sent ETH to worker %d\n", i)
	}

	wg.Wait()
	log.Printf("Sent Gas Fee to all workers\n")
}

func buildTestTx(chainID *big.Int, workerAccounts []models.Account, client *ethclient.Client, mainAccountAddress common.Address, gasPrice *big.Int) (rawTxList []models.Tx) {
	log.Printf("---------------- buildTestTx ----------------\b")

	workerWg := sync.WaitGroup{}
	numTxsWg := sync.WaitGroup{}

	rawTxMap := sync.Map{}
	for i, workerAccount := range workerAccounts {
		workerWg.Add(1)
		mainAccountNonce, err := client.NonceAt(context.Background(), mainAccountAddress, nil)
		if err != nil {
			log.Fatalf("Failed to get mainAccountNonce: %v", err)
		}
		go func(i int, mainAccountNonce *big.Int, accountHex string) {
			defer workerWg.Done()
			workerAccountNonce, err := client.NonceAt(context.Background(), workerAccount.Address, nil)
			if err != nil {
				log.Fatalf("Failed to get workerAccountNonce: %v", err)
			}
			for j := range numTxsPerWorker {
				numTxsWg.Add(1)
				go func(nonce uint64) {
					defer numTxsWg.Done()
					// 从主账户向工作账户发送ETH
					rawTx, txHash, err := buildEthTx(ethRpcUrl, workerAccount.PrivateKeyHex, chainID, big.NewInt(int64(nonce)), workerAccount.Address.String(), big.NewInt(0), big.NewInt(21000), gasPrice, false, false, nil)
					if err != nil {
						log.Printf("Failed to send ETH to worker %d: %v", i, err)
					}
					rawTxModel := models.Tx{
						From:     workerAccount.Address,
						To:       workerAccount.Address,
						Value:    big.NewInt(0),
						RawTxHex: rawTx,
						Hash:     txHash,
						Nonce:    nonce,
					}
					rawTxMap.Store(rawTxModel.From.String()+strconv.Itoa(int(rawTxModel.Nonce)), rawTxModel)
					log.Printf("Build worker test tx. Worker number: %d, Tx Number: %d\n", i, j)
				}(workerAccountNonce + j)
			}
		}(i, big.NewInt(int64(mainAccountNonce+uint64(i))), workerAccount.Address.Hex())
	}
	workerWg.Wait()
	numTxsWg.Wait()
	// 遍历所有键值对
	rawTxMap.Range(func(key, value interface{}) bool {
		rawTxList = append(rawTxList, value.(models.Tx))
		return true
	})
	return rawTxList
}

func loadWorkers(path string) (accounts []models.Account, err error) {
	log.Printf("---------------- loadWorkers ----------------\b")

	pks, err := tools.FileRead(path)
	if err != nil {
		return nil, err
	}

	for _, pk := range pks {
		spk := strings.Split(pk, ",")
		account := spk[0]
		privateKey := spk[1]
		accounts = append(accounts, models.Account{
			Address:       common.HexToAddress(account),
			Balance:       big.NewInt(0),
			PrivateKeyHex: privateKey,
		})
	}

	return
}

func broadcastTransactions(client *rpc.Client, workerAddresses []models.Tx) {
	log.Printf("---------------- broadcastTransactions ----------------\b")
	var wg sync.WaitGroup

	for i := 0; i < len(workerAddresses); i++ {
		wg.Add(1)
		go func(workerIndex int) {
			defer wg.Done()

			// 构建和广播交易
			err := client.CallContext(context.Background(), nil, "eth_sendRawTransaction", "0x"+workerAddresses[workerIndex].RawTxHex)
			if err != nil {
				log.Printf("Failed to broadcast transaction %v: %v", workerAddresses[workerIndex], err)
			}
		}(i)
	}

	wg.Wait()

	log.Printf("Broadcasted all transactions\n")
}

func buildEthTx(ethRpcUrl, privateKeyHex string, chainId, nonce *big.Int, toAddressHex string, value, gasLimit, gasPrice *big.Int, send, wait bool, wg *sync.WaitGroup) (string, string, error) {
	client, err := rpc.DialContext(context.Background(), ethRpcUrl)
	if err != nil {
		return "", "", err
	}

	ethClient := ethclient.NewClient(client)

	// 私钥，用于签署交易
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", "", fmt.Errorf("error converting private key: %s", err)
	}

	// 设置接收者的地址
	toAddress := common.HexToAddress(toAddressHex)

	// 设置数据字段，如果是标准的ETH转账，这通常是空的。
	var data []byte
	// 创建一个未签名的交易。
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce.Uint64(),
		To:       &toAddress,
		Value:    value,
		Gas:      gasLimit.Uint64(),
		GasPrice: gasPrice,
		Data:     data,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), privateKey)
	if err != nil {
		return "", "", fmt.Errorf("error signing transaction: %s", err)
	}

	// 对交易进行RLP编码，以便可以将其广播到以太坊网络。
	encodedTx, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return "", "", fmt.Errorf("error encoding transaction: %s", err)
	}

	// 将RLP编码的交易转换为十六进制字符串
	txDataHex := common.Bytes2Hex(encodedTx)

	if send {
		err = client.CallContext(context.Background(), nil, "eth_sendRawTransaction", "0x"+txDataHex)
		if err != nil {
			return "", "", err
		}

		if wait && wg != nil {
			go func(txDataHex string, txHash common.Hash, wg *sync.WaitGroup) {
				for {
					time.Sleep(2 * time.Second)
					// 补偿广播
					err = client.CallContext(context.Background(), nil, "eth_sendRawTransaction", "0x"+txDataHex)
					if err != nil && !strings.Contains(err.Error(), "already known") && !strings.Contains(err.Error(), "nonce too low") {
						log.Printf("Failed to broadcast transaction: %v", err)
						continue
					}
					// 判断是否上链
					_, pending, err := ethClient.TransactionByHash(context.Background(), txHash)
					if err != nil {
						log.Printf("Failed to get transaction by hash: %v", err)
						continue
					}
					if !pending {
						wg.Done()
						return
					}
				}
			}(txDataHex, signedTx.Hash(), wg)
		}

	}

	return txDataHex, signedTx.Hash().Hex(), nil
}
