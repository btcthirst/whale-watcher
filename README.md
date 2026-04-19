# Whale Watcher 🐋

*Read this in other languages: [English](README.md), [Українська](README.uk.md)*

A robust, real-time Go application designed to track and alert on large transactions ("whales") on the Solana blockchain.

## Features

- **Real-Time Monitoring**: Uses Solana's WebSocket `logsSubscribe` for instant event detection.
- **Enhanced Data Fetching**: Integrates with the Helius API to retrieve detailed transaction information quickly and reliably.
- **Smart Pricing Engine**: Determines token values by querying a chain of price providers (Jupiter → DexScreener → GeckoTerminal) with built-in caching for optimal performance.
- **Customizable Thresholds**: Easily configure alerts based on:
  - Minimum SOL amount
  - Minimum USD value equivalent
  - Minimum Token Amount (for tokens with unknown prices)
- **Multi-Channel Alerts**: Supports real-time notifications via **Telegram** and **Webhooks**.
- **Data Persistence**: Uses a local **SQLite** database to store processed transfers and detected whale events securely.
- **Clean Architecture**: Written in Go with modularity, testability, and high performance in mind.

## Prerequisites

- [Go](https://golang.org/doc/install) 1.21 or higher
- A Solana WebSocket endpoint (e.g., QuickNode, Alchemy)
- A [Helius API Key](https://dev.helius.xyz/)
- (Optional) Telegram Bot Token & Chat ID for notifications

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/btcthirst/whale-watcher.git
   cd whale-watcher
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Configure the environment variables. Copy the `.env.example` file to `.env`:
   ```bash
   cp .env.example .env
   ```
   
   Fill in your API keys and configure thresholds in the `.env` file:
   ```env
   TELEGRAM_BOT_TOKEN="your_telegram_bot_token"
   CHAT_ID=123456789
   HELIUS_API_KEY="your_helius_api_key"
   SOLANA_WS_URL="wss://your-solana-websocket-url"
   SOL_THRESHOLD=100
   USD_THRESHOLD=50000
   TOKEN_AMOUNT_THRESHOLD=1000000
   DB_PATH="whale.db"
   WEBHOOK_URL="https://your-webhook-endpoint.com"
   ```

## Usage

You can run the application directly or build an executable.

**Run directly:**
```bash
make run
```

**Build and execute:**
```bash
make build
./bin/whale-watcher
```

## Development & Testing

The project includes a `Makefile` to simplify common tasks:

- **Run tests**: `make test`
- **Run linters**: `make lint`
- **Format code**: `make fmt`
- **Clean build artifacts**: `make clean`

## Architecture overview

- **`cmd/watcher`**: The entry point of the application.
- **`internal/infrastructure`**: External integrations (Solana RPC, Helius API, Telegram, Webhooks).
- **`internal/application`**: Core business logic, pricing engine (`pricer`), whale detector, data deduplication, and alerting service.
- **`internal/storage/sqlite`**: Database interactions for persisting transfers and events.

## License

This project is open-source and available under the MIT License.
