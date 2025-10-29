[![Go Report Card](https://goreportcard.com/badge/trezor/blockbook)](https://goreportcard.com/report/trezor/blockbook)

# Blockbook

**Blockbook** is a back-end service for Trezor Suite. The main features of **Blockbook** are:

-   index of addresses and address balances of the connected block chain
-   fast index search
-   simple blockchain explorer
-   websocket, API and legacy Bitcore Insight compatible socket.io interfaces
-   support of multiple coins (Bitcoin and Ethereum type) with easy extensibility to other coins
-   scripts for easy creation of debian packages for backend and blockbook

## Build and installation instructions

Officially supported platform is **Debian Linux** and **AMD64** architecture.

Memory and disk requirements for initial synchronization of **Bitcoin mainnet** are around 32 GB RAM and over 180 GB of disk space. After initial synchronization, fully synchronized instance uses about 10 GB RAM.
Other coins should have lower requirements, depending on the size of their block chain. Note that fast SSD disks are highly
recommended.

User installation guide is [here](<https://wiki.trezor.io/User_manual:Running_a_local_instance_of_Trezor_Wallet_backend_(Blockbook)>).

Developer build guide is [here](/docs/build.md).

Contribution guide is [here](CONTRIBUTING.md).

### Building Blockbook for Particl

#### Prerequisites
- Go 1.23+ (will be downloaded automatically during build)
- RocksDB development libraries
- Particl daemon (particld) running and synced

#### Build Instructions

See [docs/build.md - Building Blockbook for Particl](/docs/build.md#building-blockbook-for-particl-special-case) for complete build instructions including the workaround for the `gnark-crypto` path length issue.

#### Configuration Setup

Before running Blockbook for the first time, you must create configuration files from templates:

```bash
# Copy template files
cp configs/coins/particl.json.example configs/coins/particl.json
cp particl-runtime.json.example particl-runtime.json

# Edit configuration with your RPC credentials
nano particl-runtime.json
```

Update these fields to match your `~/.particl/particl.conf`:
```json
{
  "rpc_user": "your_rpc_username",
  "rpc_pass": "your_rpc_password",
  "extended_index": true
}
```

**Note:** The `extended_index: true` setting is **critical** for Particl. It must be set in the configuration file AND you must use the `-extendedindex` command-line flag when starting Blockbook. This enables storage of Particl's privacy transaction metadata (CT fees, ring sizes, anonymous inputs/outputs).

**⚠️ IMPORTANT:** Never commit `particl.json` or `particl-runtime.json` to git! The `.gitignore` is configured to exclude them automatically. These files contain sensitive credentials that should remain private.

#### Running Blockbook

Before starting Blockbook, ensure your Particl daemon is running with RPC enabled:

```bash
# Start particld with RPC access
particld -server -rpcuser=yourusername -rpcpassword=yourpassword -addressindex=1 -spentindex=1
```

Then start Blockbook with the `-extendedindex` flag:

```bash
# From the blockbook directory
./build/blockbook \
  -blockchaincfg=particl-runtime.json \
  -datadir=/home/yourusername/.particl-blockbook/data \
  -extendedindex \
  -sync \
  -internal=:9135 \
  -public=:9235 \
  -logtostderr
```

**Important:** The `-datadir` flag specifies where Blockbook stores its RocksDB database:
- **Development/Testing**: Use `/home/yourusername/.particl-blockbook/data` (persistent, user-owned)
- **Production**: Use `/var/lib/blockbook/particl` (system-wide) or a dedicated data partition
- **NEVER use `/tmp`**: This directory is cleared on reboot and will lose all indexed data

**⚠️ CRITICAL for Particl:** The `-extendedindex` flag is **required** to properly index and display Particl's privacy features:
- **CT (Confidential Transaction) fees** - Fees embedded in privacy transactions
- **RingCT inputs** - Anonymous input types with ring signatures
- **Ring sizes** - Size of the anonymity set (e.g., 1 real input + 4 decoys = ring size 5)
- **Anon outputs** - Blinded outputs with Bulletproof range proofs
- **Value commitments** - Pedersen commitments hiding transaction amounts

Without `-extendedindex`, the explorer will only show basic Bitcoin-compatible data and **all privacy-specific fields will be missing**.

Key parameters:
- `-blockchaincfg`: Path to Particl runtime configuration JSON (use `particl-runtime.json`, NOT the template in `configs/coins/`)
- `-datadir`: Directory for Blockbook's RocksDB database (see Data Storage Locations below)
- **`-extendedindex`**: **REQUIRED** - Enables extended transaction data storage for Particl privacy features
- `-sync`: Enable blockchain synchronization
- `-internal`: Internal API port (default: 9135)
- `-public`: Public HTTP port (default: 9235)
- `-logtostderr`: Output logs to stderr instead of files
- `-certfile` and `-certkey`: Optional SSL/TLS certificates for HTTPS (use reverse proxy for production)

#### Data Storage Locations

The `-datadir` flag determines where Blockbook stores its RocksDB database (15-20GB when fully synced).

**Default (if `-datadir` not specified):** `./data` (relative to current working directory)
⚠️ **Warning:** The default relative path is **not recommended** as it changes based on where you run the command and may create the database inside your source code directory.

**Always use an absolute path with `-datadir`:**

| Environment | Recommended Path | Notes |
|------------|------------------|-------|
| **Development/Personal** | `~/.particl-blockbook/data` | User-owned, persistent, no root required |
| **Production (system-wide)** | `/var/lib/blockbook/particl` | Standard Linux location, requires setup |
| **High-volume deployments** | `/mnt/ssd/blockbook/particl` | Dedicated SSD partition for performance |
| **Testing ONLY** | `/tmp/blockbook` | ⚠️ CLEARED ON REBOOT - Never use for production! |

**Setup for production path** (`/var/lib/blockbook/particl`):
```bash
sudo mkdir -p /var/lib/blockbook/particl
sudo chown yourusername:yourusername /var/lib/blockbook/particl
# Or for dedicated blockbook user:
# sudo chown blockbook:blockbook /var/lib/blockbook/particl
```

#### Initial Synchronization

The initial sync can take several hours depending on:
- Particl blockchain size
- Your hardware (SSD recommended)
- Available RAM (8GB+ recommended)

Monitor progress in the logs or via the internal API: `http://localhost:9030`

#### Web Deployment

For production deployment on a web server:

1. **Generate SSL certificates** (Let's Encrypt recommended):
   ```bash
   certbot certonly --standalone -d explorer.yourdomain.com
   ```

2. **Configure reverse proxy** (nginx example):
   ```nginx
   server {
       listen 443 ssl http2;
       server_name explorer.yourdomain.com;

       ssl_certificate /etc/letsencrypt/live/explorer.yourdomain.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/explorer.yourdomain.com/privkey.pem;

       location / {
           proxy_pass https://localhost:9130;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }

       # WebSocket support
       location /websocket {
           proxy_pass https://localhost:9130/websocket;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection "upgrade";
       }
   }
   ```

3. **Run Blockbook as a systemd service**:
   ```ini
   [Unit]
   Description=Particl Blockbook
   After=network.target particld.service
   Requires=particld.service

   [Service]
   Type=simple
   User=blockbook
   ExecStart=/opt/blockbook/build/blockbook \
       -blockchaincfg=/opt/blockbook/particl-runtime.json \
       -datadir=/var/lib/blockbook/particl \
       -extendedindex \
       -sync \
       -internal=:9030 \
       -public=:9130 \
       -certfile=/etc/letsencrypt/live/explorer.yourdomain.com/fullchain.pem \
       -certkey=/etc/letsencrypt/live/explorer.yourdomain.com/privkey.pem
   Restart=always
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```

4. **Enable and start the service**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable blockbook-particl
   sudo systemctl start blockbook-particl
   ```

Access your explorer at `https://explorer.yourdomain.com`

## Implemented coins

Blockbook currently supports over 30 coins. The Trezor team implemented

-   Bitcoin, Bitcoin Cash, Zcash, Dash, Litecoin, Bitcoin Gold, Ethereum, Ethereum Classic, Dogecoin, Namecoin, Vertcoin, DigiByte, Liquid

the rest of coins were implemented by the community.

Testnets for some coins are also supported, for example:

-   Bitcoin Testnet, Bitcoin Cash Testnet, ZCash Testnet, Ethereum Testnets (Sepolia, Hoodi)

### Particl Integration

**Particl** has been integrated using the RPC-based approach with full support for:
- **Standard transactions** (Segwit-compatible)
- **Confidential Transactions (CT)** with Bulletproofs
- **RingCT (anonymous) transactions** with ring signatures
- Native Particl addressing (P-addresses for public, ps-addresses for stealth)

Particl uses the RPC backend similar to Zcash, fetching parsed transaction data from the Particl daemon rather than parsing raw block data. This approach ensures accurate handling of Particl's privacy features.

#### Testing Particl Integration

The Particl integration includes comprehensive test coverage for all transaction types and privacy features:

**Unit Tests** (`bchain/coins/part/particlparser_test.go`)
- **Location**: Co-located with Particl parser implementation
- **Test Count**: 10 test functions covering 622 lines
- **Coverage**: Address parsing (P2PKH, P2CS, P2SH256), transaction serialization (PackTx/UnpackTx), privacy transactions (CT, RingCT), data outputs
- **Run**: `go test -v -count=1 -tags=unittest ./bchain/coins/part`

**Integration Tests** (`tests/tests.json`)
- **Configuration**: Lines 159-163 in central test registry
- **Coverage**: 10 RPC tests (GetBlock, GetTransaction, MempoolSync, etc.), 3 sync tests (ConnectBlocksParallel, ConnectBlocks, HandleFork)
- **Requirements**: Running `particld` daemon with RPC enabled
- **Run**: `go test -v -count=1 -tags=integration ./tests -run='TestIntegration/particl' -timeout 10m`

**UI/Template Tests** (`bchain/coins/part/test_ui.sh`)
- **Purpose**: Verify web interface correctly displays all Particl transaction types
- **Test Cases**: 6 automated tests
  - CT (Confidential) transactions show "Blinded" instead of "0 PART"
  - RingCT (Anonymous) transactions display ring size and anonymity info
  - Standard transactions show full PART amounts
  - Cold Staking (P2CS) addresses render correctly (prefixes: "2", "33")
  - Address pages load with balances
  - API endpoints function correctly
- **Latest Results**: 6/6 PASSING (100% success rate)
- **Run**: `bash bchain/coins/part/test_ui.sh` (requires Blockbook running on http://localhost:9131)

**Real Blockchain Test Data**:
All tests use actual mainnet transactions to ensure accuracy:
- **CT Transaction**: `4be9ec51111a27794b5c3ea1fe58a2658f58a354595a73d55cffb8394f07ad48` (block 2,028,364)
- **RingCT Transaction**: `f48d5bce842ac718b2995642ebf2fe35cbe70f10e92069e21d9959dcd6df7384` (block 488,901)
- **Standard Transaction**: `82fa17dfb9c5fb91d8fcc89674756c5df86d743ca7d20e58977926a648fb37f8` (block 2,028,000)
- **Cold Staking**: `819e21b3b1d12539371df023b0865da74648f342c912b535f639ca622d97abf5` (block 2,027,702)

List of all implemented coins is in [the registry of ports](/docs/ports.md).

## Common issues when running Blockbook or implementing additional coins

#### Out of memory when doing initial synchronization

How to reduce memory footprint of the initial sync:

-   disable rocksdb cache by parameter `-dbcache=0`, the default size is 500MB
-   run blockbook with parameter `-workers=1`. This disables bulk import mode, which caches a lot of data in memory (not in rocksdb cache). It will run about twice as slowly but especially for smaller blockchains it is no problem at all.

Please add your experience to this [issue](https://github.com/trezor/blockbook/issues/43).

#### Error `internalState: database is in inconsistent state and cannot be used`

Blockbook was killed during the initial import, most commonly by OOM killer.
By default, Blockbook performs the initial import in bulk import mode, which for performance reasons does not store all data immediately to the database. If Blockbook is killed during this phase, the database is left in an inconsistent state.

See above how to reduce the memory footprint, delete the database files and run the import again.

Check [this](https://github.com/trezor/blockbook/issues/89) or [this](https://github.com/trezor/blockbook/issues/147) issue for more info.

#### Running on Ubuntu

[This issue](https://github.com/trezor/blockbook/issues/45) discusses how to run Blockbook on Ubuntu. If you have some additional experience with Blockbook on Ubuntu, please add it to [this issue](https://github.com/trezor/blockbook/issues/45).

#### My coin implementation is reporting parse errors when importing blockchain

Your coin's block/transaction data may not be compatible with `BitcoinParser` `ParseBlock`/`ParseTx`, which is used by default. In that case, implement your coin in a similar way we used in case of [zcash](https://github.com/trezor/blockbook/tree/master/bchain/coins/zec) and some other coins. The principle is not to parse the block/transaction data in Blockbook but instead to get parsed transactions as json from the backend.

## Data storage in RocksDB

Blockbook stores data the key-value store RocksDB. Database format is described [here](/docs/rocksdb.md).

## API

Blockbook API is described [here](/docs/api.md).

## Environment variables

List of environment variables that affect Blockbook's behavior is [here](/docs/env.md).
