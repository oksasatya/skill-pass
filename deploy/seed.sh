#!/usr/bin/env bash
# Deploy SkillPassCertificate to local anvil and issue 2 test certificates.
# Run after `make dev-up` from the repo root.
set -euo pipefail

export PATH="$HOME/.foundry/bin:$PATH"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RPC_URL="http://localhost:8545"
OWNER_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
CONTRACT="0x5FbDB2315678afecb367f032d93F642f64180aa3"

echo "==> Deploying SkillPassCertificate (account[0] nonce=0)..."
cd "$REPO_ROOT/contracts"
PRIVATE_KEY="$OWNER_KEY" forge script script/Deploy.s.sol:Deploy \
  --rpc-url "$RPC_URL" \
  --broadcast \
  --quiet

echo "==> Contract should be at: $CONTRACT"
CODE=$(cast code "$CONTRACT" --rpc-url "$RPC_URL")
if [ "$CODE" = "0x" ]; then
  echo "ERROR: no code at $CONTRACT — deploy failed or address mismatch" >&2
  exit 1
fi
echo "    Code confirmed at $CONTRACT"

echo "==> Issuing certificate #0 (recipient: account[1])..."
cast send "$CONTRACT" \
  "issueCertificate(address,string,string,string,string,string)" \
  "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" \
  "Full Stack Web3" \
  "Oksa Satya" \
  "SkillPass Academy" \
  "Completed the Full Stack Web3 program" \
  "ipfs://demo1" \
  --rpc-url "$RPC_URL" \
  --private-key "$OWNER_KEY" \
  --quiet

echo "==> Issuing certificate #1 (recipient: account[2])..."
cast send "$CONTRACT" \
  "issueCertificate(address,string,string,string,string,string)" \
  "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC" \
  "Smart Contract Security" \
  "Budi Santoso" \
  "SkillPass Academy" \
  "Completed the Smart Contract Security audit course" \
  "ipfs://demo2" \
  --rpc-url "$RPC_URL" \
  --private-key "$OWNER_KEY" \
  --quiet

echo "==> Seed complete — 2 certificates issued."
