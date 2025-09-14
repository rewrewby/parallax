# Parallax

Official Golang implementation of the **Parallax protocol**.

> **Parallax** is an open experiment in programmable money. It combines Bitcoin’s fixed monetary rules with Ethereum’s virtual machine to deliver a scarce, decentralized, and programmable timechain.

---

## What is Parallax?

- ⛏️ **Proof of Work (Ethash)** — memory-hard, GPU-friendly mining for broad participation.  
- 🕒 **10-minute block interval** — stability and probabilistic finality inspired by Bitcoin’s timechain.  
- 💰 **21M fixed supply** — halving cycles every 210,000 blocks; no premine; no privileged allocations.  
- ⚙️ **EVM execution** — Solidity & Vyper smart contracts; compatible with Ethereum tooling.  
- 🌐 **Neutral & community-driven** — no governance over supply; protocol stewardship trends to the community.

Parallax is **not** a replacement for Bitcoin. It is a complementary system that explores what becomes possible when **Bitcoin’s monetary discipline** meets **Ethereum’s expressiveness**.

---

## System Parameters

| Parameter                     | Value                                 |
|------------------------------|---------------------------------------|
| Consensus Mechanism          | Proof of Work (**Ethash**)            |
| Target Block Interval        | **600 seconds** (10 minutes)          |
| Difficulty Retarget          | **2016 blocks** (~2 weeks)            |
| Initial Block Reward         | **50** coins                          |
| Halving Interval             | **210,000** blocks (~4 years)         |
| Maximum Supply               | **21,000,000** coins                  |
| Premine                      | **0**                                  |
| Execution Environment        | **EVM** (account-based)               |
| Fee Model                    | **First-price auction** (no burn)     |
| Block Gas Limit (initial)    | **600M** gas; ±0.1% elastic per block |
| Coinbase Maturity            | **100 blocks**                         |

---

## Building from Source

Parallax requires **Go 1.25+** and a C compiler.

```bash
make prlx
```

Build the full suite:

```bash
make all
```

---

## Executables

Binaries are located under `build/bin`:

| Command        | Description |
|----------------|-------------|
| **`prlx`** | Main CLI client. Runs full, archive, or light nodes; exposes JSON-RPC over HTTP, WS, and IPC. |
| `clef`         | Stand-alone signer for secure account operations. |
| `devp2p`       | Networking utilities to inspect and interact at the P2P layer. |
| `abigen`       | Generates type-safe Go bindings from contract ABIs. |
| `bootnode`     | Lightweight discovery node to bootstrap networks. |
| `evm`          | Execute and debug EVM bytecode snippets in isolation. |
| `rlpdump`      | Decode RLP structures into a human-readable form. |
| `puppeth`      | Wizard to create and manage custom Parallax networks. |

---

## Running a Node

> Mainnet is not yet live. The node will default to testnet instead.

Mainnet (interactive console):

```bash
prlx console
```

Testnet:

```bash
prlx --testnet console
```

### Hardware Recommendations

- **Minimum**: 2 cores, 4 GB RAM, 500 GB SSD, 8 Mbps  
- **Recommended**: 4+ cores, 16 GB RAM, 1 TB SSD, 25+ Mbps

---

## Mining Parallax coins

Parallax miner can be started with:

```
prlx --miner.coinbase <YOUR_WALLET_ADDRESS> --miner.threads 1 --mine
```

You can use your existing web3 wallet address from MetaMask or any other Ethereum based wallet. Or create a new wallet address using either `clef` or `parallaxkey`:

```
parallaxkey generate
```

```
clef newaccount
```

---

## JSON-RPC (Developers)

IPC is enabled by default. Enable HTTP/WS explicitly:

```
--http --http.addr 0.0.0.0 --http.port 8545 --http.api eth,net,web3
--ws   --ws.addr   0.0.0.0 --ws.port   8546 --ws.api   eth,net,web3
```

> ⚠️ **Security**: Do **not** expose RPC to the public Internet. Use firewalls, auth proxies, and restricted origins.

---

## Philosophy & Governance

- **Fair launch** — no premine, no insider allocations; everyone starts at genesis.  
- **Immutable monetary policy** — 21M hard cap with predictable halving; no fee burn.  
- **Open participation** — Ethash favors decentralized, commodity hardware mining.  
- **Community stewardship** — developed under MicroStack initially; long-term ownership transitions to the community.  
- **Neutrality first** — monetary rules are not subject to governance or discretion.

---

## Contribution

We welcome contributions aligned with **neutrality, openness, and decentralization**.

1. Fork the repo
2. Implement your changes
3. Open a PR against `main`

**Guidelines**

- Format with `gofmt`; document public symbols following Go conventions.  
- Keep commits focused; prefix messages with affected packages (e.g., `prlx, rpc:`).  

---

## License

- **Library code** (`/` excluding `cmd/`): [LGPL v3](https://www.gnu.org/licenses/lgpl-3.0.en.html)  
- **Executables** (`/cmd/*`): [GPL v3](https://www.gnu.org/licenses/gpl-3.0.en.html)

---

> ⚡ **Parallax is an open experiment.** Its future is written by builders, miners, and users—not by any single company or foundation.
