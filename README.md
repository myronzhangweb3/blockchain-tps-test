# BlockChain TPS Test

This project tests blockchain transaction throughput (TPS) by sending multiple concurrent transactions from different accounts.

## Test Principles

1. Account Generation
   - The system first generates multiple test accounts using `generate-account`
   
2. Transaction Testing
   - Each account is funded with a minimum balance (0.01 ETH) from a main account
   - The main account's private key is configured in `.env`
   - Multiple worker goroutines (default 30) send transactions concurrently
   - Each worker processes a configured number of transactions (default 10)
   - Transactions are simple ETH transfers between accounts
   - Gas price is calculated dynamically with a configurable multiplier
   - System monitors and replenishes gas fees when account balance is low

## Usage

### Local


#### Environment

```bash
cp .env.example .env
```

#### Generate account

```bash
go run cmd/generate_account/main.go
```

#### Send EOA tx

```bash
go run cmd/send_eoa_tx/main.go
```

### Docker

#### Environment

```bash
cp .env.example .env.docker
cp docker-compose.yml.example docker-compose.yml
```

#### Generate account

```bash
docker-compose up generate-account --build
```

#### Send EOA tx

```bash
docker-compose up send-eoa-tx --build
```