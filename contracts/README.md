# SkillPassCertificate — Foundry

## Setup

1. Install Foundry: `curl -L https://foundry.paradigm.xyz | bash && foundryup`
2. Install dependencies (not vendored — run after cloning):
   ```
   make install
   ```
   Or manually: `forge install OpenZeppelin/openzeppelin-contracts@v5.1.0 --no-git && forge install foundry-rs/forge-std --no-git`
3. `cp .env.example .env` and fill in a **throwaway testnet** `PRIVATE_KEY`,
   `BASE_SEPOLIA_RPC_URL`, and `BASESCAN_API_KEY`. Never commit `.env`.
4. Fund the deployer with Base Sepolia ETH from a faucet.

## Test

    make test
    # or: forge test -vvv

## Deploy to Base Sepolia (chainId 84532)

    source .env
    forge script script/Deploy.s.sol:Deploy \
      --rpc-url base_sepolia --broadcast --verify -vvv

## Export ABI (for the frontend in Phase 2)

    jq '.abi' out/SkillPassCertificate.sol/SkillPassCertificate.json \
      > ../deployments/SkillPassCertificate.abi.json

Record the deployed address in `../deployments/base-sepolia.json`.
