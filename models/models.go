package models

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Account struct {
	Address       common.Address
	Balance       *big.Int
	PrivateKeyHex string
}

type Tx struct {
	From     common.Address
	To       common.Address
	Value    *big.Int
	Nonce    uint64
	RawTxHex string
	Hash     string
}
