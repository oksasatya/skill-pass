# SkillPass Phase 1 — Smart Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build, test, and deploy the `SkillPassCertificate` soulbound ERC-721 contract to Base Sepolia with Foundry.

**Architecture:** A single OpenZeppelin Contracts 5.x `ERC721` + `Ownable` contract. Only the owner can issue (mint) certificates. Certificates are non-transferable (soulbound) via an `_update` override and reverting `approve`/`setApprovalForAll`; ERC-5192 signals the lock. Per-certificate data is stored on-chain; `tokenURI` returns the optional stored metadata URI. Owner enumeration is intentionally NOT on-chain (done off-chain from events in later phases).

**Tech Stack:** Solidity `^0.8.24`, Foundry (forge/cast/anvil), OpenZeppelin Contracts 5.x, Base Sepolia testnet.

## Global Constraints

- Solidity pragma `^0.8.24`; `foundry.toml` pins `solc = "0.8.24"`.
- OpenZeppelin Contracts **5.x** only. Use 5.x APIs: `_update`, `_ownerOf`, `_requireOwned`, `_safeMint`. Do NOT use `Counters`, `_exists`, `_beforeTokenTransfer`, `_afterTokenTransfer` (removed/changed in 5.x).
- `Ownable` requires `Ownable(initialOwner)` in the constructor.
- Soulbound check uses the **previous owner**: revert when `from != address(0) && to != address(0)` (allow mint where `from == 0` and burn where `to == 0`). Do NOT key the check on `auth`.
- Gas-cheap **custom errors** (no string reverts): `Soulbound()`, `ApprovalDisabled()`, `ZeroRecipient()`, `StringTooLong()`.
- TDD (§16: yes for this contract): write the failing Foundry test first, then the minimal implementation.
- Security: deployer uses a **throwaway testnet-only wallet**; `PRIVATE_KEY` lives in `contracts/.env`, which MUST be git-ignored. Repo is public.
- Network: Base Sepolia, `chainId 84532`. Contract name `"SkillPass Certificate"`, symbol `"SKILL"`.
- String length caps (constants): `MAX_TITLE = 200`, `MAX_NAME = 100`, `MAX_DESC = 1000`, `MAX_URI = 300`.

---

### Task 1: Scaffold Foundry project + OpenZeppelin + compiling skeleton

**Files:**
- Create: `contracts/foundry.toml`
- Create: `contracts/remappings.txt`
- Create: `contracts/.gitignore`
- Create: `contracts/.env.example`
- Create: `contracts/src/SkillPassCertificate.sol` (skeleton)
- Create: root `.gitignore`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `SkillPassCertificate(address initialOwner)` constructor; an empty-but-compiling contract that later tasks extend.

- [ ] **Step 1: Initialize Foundry in `contracts/`**

```bash
cd /Volumes/Project/skill-pass
forge init contracts --no-git --no-commit
rm contracts/src/Counter.sol contracts/test/Counter.t.sol contracts/script/Counter.s.sol
```

- [ ] **Step 2: Install OpenZeppelin Contracts 5.x**

```bash
cd /Volumes/Project/skill-pass/contracts
forge install OpenZeppelin/openzeppelin-contracts@v5.1.0 --no-git --no-commit
```

- [ ] **Step 3: Write `contracts/remappings.txt`**

```
@openzeppelin/contracts/=lib/openzeppelin-contracts/contracts/
forge-std/=lib/forge-std/src/
```

- [ ] **Step 4: Write `contracts/foundry.toml`**

```toml
[profile.default]
src = "src"
out = "out"
libs = ["lib"]
solc = "0.8.24"
optimizer = true
optimizer_runs = 200

[rpc_endpoints]
base_sepolia = "${BASE_SEPOLIA_RPC_URL}"

[etherscan]
base_sepolia = { key = "${BASESCAN_API_KEY}", url = "https://api-sepolia.basescan.org/api", chain = 84532 }
```

- [ ] **Step 5: Write `contracts/.gitignore`**

```
out/
cache/
broadcast/
.env
```

- [ ] **Step 6: Write `contracts/.env.example`**

```
# Throwaway TESTNET wallet only — never a key holding real funds
PRIVATE_KEY=0x...
BASE_SEPOLIA_RPC_URL=https://sepolia.base.org
BASESCAN_API_KEY=
```

- [ ] **Step 7: Write root `.gitignore`**

```
# Node
node_modules/
dist/
# Foundry (also covered in contracts/.gitignore)
contracts/out/
contracts/cache/
contracts/broadcast/
# Env / secrets
.env
.env.*
!.env.example
# Go (later phases)
/services/**/bin/
```

- [ ] **Step 8: Write the skeleton `contracts/src/SkillPassCertificate.sol`**

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC721} from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

/// @title SkillPassCertificate
/// @notice Soulbound (non-transferable) ERC-721 certificate. Owner-only issuance.
contract SkillPassCertificate is ERC721, Ownable {
    constructor(address initialOwner)
        ERC721("SkillPass Certificate", "SKILL")
        Ownable(initialOwner)
    {}
}
```

- [ ] **Step 9: Verify it compiles**

Run: `cd /Volumes/Project/skill-pass/contracts && forge build`
Expected: `Compiler run successful`

- [ ] **Step 10: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/foundry.toml contracts/remappings.txt contracts/.gitignore contracts/.env.example contracts/src/SkillPassCertificate.sol .gitignore contracts/lib .gitmodules
git commit -m "chore(contracts): scaffold Foundry + OpenZeppelin 5.x skeleton"
```

---

### Task 2: Issue certificate + storage + read (happy path)

**Files:**
- Modify: `contracts/src/SkillPassCertificate.sol`
- Test: `contracts/test/SkillPassCertificate.t.sol`

**Interfaces:**
- Consumes: `SkillPassCertificate(address)` constructor from Task 1.
- Produces:
  - `struct Certificate { string title; string recipientName; string issuerName; string description; string metadataURI; uint256 issuedAt; }`
  - `function issueCertificate(address recipient, string calldata title, string calldata recipientName, string calldata issuerName, string calldata description, string calldata metadataURI) external returns (uint256 tokenId)`
  - `function getCertificate(uint256 tokenId) external view returns (Certificate memory cert, address recipient)`
  - `function totalSupply() external view returns (uint256)`
  - `event CertificateIssued(uint256 indexed tokenId, address indexed recipient, string title, string issuerName, uint256 issuedAt)`

- [ ] **Step 1: Write the failing test**

Create `contracts/test/SkillPassCertificate.t.sol`:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {SkillPassCertificate} from "../src/SkillPassCertificate.sol";

contract SkillPassCertificateTest is Test {
    SkillPassCertificate internal cert;
    address internal owner = address(0xA11CE);
    address internal recipient = address(0xB0B);

    function setUp() public {
        vm.prank(owner);
        cert = new SkillPassCertificate(owner);
    }

    function test_IssueCertificate_StoresDataAndMints() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(
            recipient, "Full Stack Web3", "Oksa Satya", "SkillPass Academy", "Completed program", "ipfs://abc"
        );

        assertEq(tokenId, 1);
        assertEq(cert.ownerOf(tokenId), recipient);
        assertEq(cert.totalSupply(), 1);

        (SkillPassCertificate.Certificate memory c, address rcpt) = cert.getCertificate(tokenId);
        assertEq(c.title, "Full Stack Web3");
        assertEq(c.recipientName, "Oksa Satya");
        assertEq(c.issuerName, "SkillPass Academy");
        assertEq(c.description, "Completed program");
        assertEq(c.metadataURI, "ipfs://abc");
        assertEq(rcpt, recipient);
        assertGt(c.issuedAt, 0);
    }

    function test_IssueCertificate_EmitsEvent() public {
        vm.expectEmit(true, true, false, true);
        emit SkillPassCertificate.CertificateIssued(1, recipient, "Full Stack Web3", "SkillPass Academy", block.timestamp);
        vm.prank(owner);
        cert.issueCertificate(recipient, "Full Stack Web3", "Oksa Satya", "SkillPass Academy", "Completed program", "ipfs://abc");
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-contract SkillPassCertificateTest -vvv`
Expected: FAIL — compile error (`issueCertificate`, `getCertificate`, `totalSupply`, `Certificate`, `CertificateIssued` not defined).

- [ ] **Step 3: Implement the minimal code**

Edit `contracts/src/SkillPassCertificate.sol` — add inside the contract body:

```solidity
    struct Certificate {
        string title;
        string recipientName;
        string issuerName;
        string description;
        string metadataURI;
        uint256 issuedAt;
    }

    event CertificateIssued(
        uint256 indexed tokenId,
        address indexed recipient,
        string title,
        string issuerName,
        uint256 issuedAt
    );

    uint256 private _nextTokenId = 1;
    mapping(uint256 tokenId => Certificate) private _certificates;

    function issueCertificate(
        address recipient,
        string calldata title,
        string calldata recipientName,
        string calldata issuerName,
        string calldata description,
        string calldata metadataURI
    ) external onlyOwner returns (uint256 tokenId) {
        tokenId = _nextTokenId++;
        _certificates[tokenId] = Certificate({
            title: title,
            recipientName: recipientName,
            issuerName: issuerName,
            description: description,
            metadataURI: metadataURI,
            issuedAt: block.timestamp
        });
        emit CertificateIssued(tokenId, recipient, title, issuerName, block.timestamp);
        _safeMint(recipient, tokenId);
    }

    function getCertificate(uint256 tokenId)
        external
        view
        returns (Certificate memory cert, address recipient)
    {
        _requireOwned(tokenId);
        return (_certificates[tokenId], _ownerOf(tokenId));
    }

    function totalSupply() external view returns (uint256) {
        return _nextTokenId - 1;
    }
```

> Note: `onlyOwner` is added now (constructor already wires `Ownable`); Task 3 tests the access-control behavior explicitly.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-contract SkillPassCertificateTest -vvv`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/src/SkillPassCertificate.sol contracts/test/SkillPassCertificate.t.sol
git commit -m "feat(contracts): issue certificate with on-chain storage and event"
```

---

### Task 3: Access control — only owner can issue

**Files:**
- Modify: `contracts/test/SkillPassCertificate.t.sol`

**Interfaces:**
- Consumes: `issueCertificate(...)` from Task 2; `Ownable.OwnableUnauthorizedAccount(address)` error from OZ 5.x.
- Produces: no new contract surface (verifies existing `onlyOwner`).

- [ ] **Step 1: Write the failing test**

Add the import at the top of the test file (below the existing imports):

```solidity
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
```

Add inside `SkillPassCertificateTest`:

```solidity
    function test_IssueCertificate_RevertWhen_NotOwner() public {
        address attacker = address(0xBAD);
        vm.prank(attacker);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, attacker));
        cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");
    }
```

- [ ] **Step 2: Run test to verify it fails or passes**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-test test_IssueCertificate_RevertWhen_NotOwner -vvv`
Expected: PASS — `onlyOwner` was added in Task 2, so this test confirms the guard. (If it FAILS, `onlyOwner` is missing from `issueCertificate` — add it.)

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/test/SkillPassCertificate.t.sol
git commit -m "test(contracts): assert only owner can issue certificate"
```

---

### Task 4: Input validation — zero recipient + string length caps

**Files:**
- Modify: `contracts/src/SkillPassCertificate.sol`
- Modify: `contracts/test/SkillPassCertificate.t.sol`

**Interfaces:**
- Consumes: `issueCertificate(...)` from Task 2.
- Produces: errors `ZeroRecipient()`, `StringTooLong()`; constants `MAX_TITLE`, `MAX_NAME`, `MAX_DESC`, `MAX_URI`.

- [ ] **Step 1: Write the failing tests**

Add inside `SkillPassCertificateTest`:

```solidity
    function test_IssueCertificate_RevertWhen_ZeroRecipient() public {
        vm.prank(owner);
        vm.expectRevert(SkillPassCertificate.ZeroRecipient.selector);
        cert.issueCertificate(address(0), "X", "Y", "Z", "D", "ipfs://x");
    }

    function test_IssueCertificate_RevertWhen_TitleTooLong() public {
        string memory longTitle = string(new bytes(201)); // 201 bytes > MAX_TITLE (200)
        vm.prank(owner);
        vm.expectRevert(SkillPassCertificate.StringTooLong.selector);
        cert.issueCertificate(recipient, longTitle, "Y", "Z", "D", "ipfs://x");
    }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-test "test_IssueCertificate_RevertWhen_(ZeroRecipient|TitleTooLong)" -vvv`
Expected: FAIL — compile error (`ZeroRecipient`, `StringTooLong` not defined).

- [ ] **Step 3: Implement validation**

In `contracts/src/SkillPassCertificate.sol`, add the errors + constants near the top of the contract body:

```solidity
    error ZeroRecipient();
    error StringTooLong();

    uint256 private constant MAX_TITLE = 200;
    uint256 private constant MAX_NAME = 100;
    uint256 private constant MAX_DESC = 1000;
    uint256 private constant MAX_URI = 300;
```

Add the guards at the very start of `issueCertificate` (before `tokenId = _nextTokenId++;`):

```solidity
        if (recipient == address(0)) revert ZeroRecipient();
        if (bytes(title).length > MAX_TITLE) revert StringTooLong();
        if (bytes(recipientName).length > MAX_NAME) revert StringTooLong();
        if (bytes(issuerName).length > MAX_NAME) revert StringTooLong();
        if (bytes(description).length > MAX_DESC) revert StringTooLong();
        if (bytes(metadataURI).length > MAX_URI) revert StringTooLong();
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-contract SkillPassCertificateTest -vvv`
Expected: PASS (all tests so far).

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/src/SkillPassCertificate.sol contracts/test/SkillPassCertificate.t.sol
git commit -m "feat(contracts): validate recipient and string lengths on issue"
```

---

### Task 5: Soulbound — block transfers and approvals

**Files:**
- Modify: `contracts/src/SkillPassCertificate.sol`
- Modify: `contracts/test/SkillPassCertificate.t.sol`

**Interfaces:**
- Consumes: `issueCertificate(...)`; OZ `ERC721._update`, `approve`, `setApprovalForAll`.
- Produces: errors `Soulbound()`, `ApprovalDisabled()`; `_update` override; reverting `approve` / `setApprovalForAll`.

- [ ] **Step 1: Write the failing tests**

Add inside `SkillPassCertificateTest`:

```solidity
    function test_Transfer_RevertWhen_Soulbound() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");

        vm.prank(recipient);
        vm.expectRevert(SkillPassCertificate.Soulbound.selector);
        cert.transferFrom(recipient, address(0xCAFE), tokenId);
    }

    function test_SafeTransfer_RevertWhen_Soulbound() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");

        vm.prank(recipient);
        vm.expectRevert(SkillPassCertificate.Soulbound.selector);
        cert.safeTransferFrom(recipient, address(0xCAFE), tokenId);
    }

    function test_SafeTransferWithData_RevertWhen_Soulbound() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");

        vm.prank(recipient);
        vm.expectRevert(SkillPassCertificate.Soulbound.selector);
        cert.safeTransferFrom(recipient, address(0xCAFE), tokenId, "");
    }

    function test_Approve_RevertWhen_Disabled() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");

        vm.prank(recipient);
        vm.expectRevert(SkillPassCertificate.ApprovalDisabled.selector);
        cert.approve(address(0xCAFE), tokenId);
    }

    function test_SetApprovalForAll_RevertWhen_Disabled() public {
        vm.prank(recipient);
        vm.expectRevert(SkillPassCertificate.ApprovalDisabled.selector);
        cert.setApprovalForAll(address(0xCAFE), true);
    }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-test "RevertWhen_(Soulbound|Disabled)" -vvv`
Expected: FAIL — `Soulbound` / `ApprovalDisabled` not defined; transfers currently succeed.

- [ ] **Step 3: Implement soulbound + approval blocks**

In `contracts/src/SkillPassCertificate.sol`, add the errors near the other errors:

```solidity
    error Soulbound();
    error ApprovalDisabled();
```

Add these overrides inside the contract body:

```solidity
    /// @dev Block transfers; allow mint (from == 0) and burn (to == 0).
    function _update(address to, uint256 tokenId, address auth)
        internal
        override
        returns (address)
    {
        address from = _ownerOf(tokenId);
        if (from != address(0) && to != address(0)) revert Soulbound();
        return super._update(to, tokenId, auth);
    }

    function approve(address, uint256) public pure override {
        revert ApprovalDisabled();
    }

    function setApprovalForAll(address, bool) public pure override {
        revert ApprovalDisabled();
    }
```

- [ ] **Step 4: Run the full suite to verify it passes**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-contract SkillPassCertificateTest -vvv`
Expected: PASS (all tests, including the earlier mint test — minting still works because `from == address(0)` on mint).

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/src/SkillPassCertificate.sol contracts/test/SkillPassCertificate.t.sol
git commit -m "feat(contracts): make certificate soulbound and block approvals"
```

---

### Task 6: ERC-5192 lock signal + tokenURI

**Files:**
- Modify: `contracts/src/SkillPassCertificate.sol`
- Modify: `contracts/test/SkillPassCertificate.t.sol`

**Interfaces:**
- Consumes: `issueCertificate(...)`; OZ `ERC721.supportsInterface`, `tokenURI`, `_requireOwned`.
- Produces: `event Locked(uint256 tokenId)`; `function locked(uint256) external view returns (bool)`; `tokenURI` override; `supportsInterface` advertising `0xb45a3c0e`.

- [ ] **Step 1: Write the failing tests**

Add inside `SkillPassCertificateTest`:

```solidity
    function test_Locked_ReturnsTrue() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");
        assertTrue(cert.locked(tokenId));
    }

    function test_SupportsInterface_ERC5192() public view {
        assertTrue(cert.supportsInterface(0xb45a3c0e)); // ERC-5192
        assertTrue(cert.supportsInterface(0x80ac58cd)); // ERC-721
    }

    function test_TokenURI_ReturnsStoredURI() public {
        vm.prank(owner);
        uint256 tokenId = cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://meta");
        assertEq(cert.tokenURI(tokenId), "ipfs://meta");
    }

    function test_EmitsLockedOnMint() public {
        vm.expectEmit(false, false, false, true);
        emit SkillPassCertificate.Locked(1);
        vm.prank(owner);
        cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");
    }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Volumes/Project/skill-pass/contracts && forge test --match-test "(Locked|SupportsInterface_ERC5192|TokenURI_ReturnsStoredURI)" -vvv`
Expected: FAIL — `locked`, `Locked`, the interface id, and `tokenURI` override not yet present.

- [ ] **Step 3: Implement ERC-5192 + tokenURI**

In `contracts/src/SkillPassCertificate.sol`, add the event near the other event:

```solidity
    event Locked(uint256 tokenId); // ERC-5192
```

Emit it inside `issueCertificate` immediately after the `CertificateIssued` emit:

```solidity
        emit Locked(tokenId);
```

Add these functions inside the contract body:

```solidity
    function locked(uint256 tokenId) external view returns (bool) {
        _requireOwned(tokenId);
        return true;
    }

    function tokenURI(uint256 tokenId) public view override returns (string memory) {
        _requireOwned(tokenId);
        return _certificates[tokenId].metadataURI;
    }

    function supportsInterface(bytes4 interfaceId) public view override returns (bool) {
        return interfaceId == 0xb45a3c0e || super.supportsInterface(interfaceId);
    }
```

- [ ] **Step 4: Run the full suite + formatter**

Run: `cd /Volumes/Project/skill-pass/contracts && forge fmt && forge test --match-contract SkillPassCertificateTest -vvv`
Expected: PASS (all tests). The assembled contract now matches the spec §6.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/src/SkillPassCertificate.sol contracts/test/SkillPassCertificate.t.sol
git commit -m "feat(contracts): add ERC-5192 lock signal and tokenURI"
```

---

### Task 7: Deploy script + ABI/address export

**Files:**
- Create: `contracts/script/Deploy.s.sol`
- Create: `contracts/README.md` (deploy + verify runbook)
- Create: `deployments/.gitkeep`

**Interfaces:**
- Consumes: `SkillPassCertificate(address initialOwner)` constructor.
- Produces: a `Deploy` script with `run()`; a documented deploy + verify + ABI-export procedure.

- [ ] **Step 1: Write the deploy script**

Create `contracts/script/Deploy.s.sol`:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console2} from "forge-std/Script.sol";
import {SkillPassCertificate} from "../src/SkillPassCertificate.sol";

contract Deploy is Script {
    function run() external returns (SkillPassCertificate cert) {
        uint256 pk = vm.envUint("PRIVATE_KEY");
        address owner = vm.addr(pk);

        vm.startBroadcast(pk);
        cert = new SkillPassCertificate(owner);
        vm.stopBroadcast();

        console2.log("SkillPassCertificate deployed at:", address(cert));
        console2.log("Owner:", owner);
    }
}
```

- [ ] **Step 2: Verify the script compiles + simulate against a local fork**

```bash
cd /Volumes/Project/skill-pass/contracts
forge build
anvil &                 # local chain in another shell, or use --fork-url base_sepolia
PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  forge script script/Deploy.s.sol:Deploy --rpc-url http://localhost:8545
```
Expected: simulation succeeds, logs a deployed address. (The key above is Anvil's well-known test key — local only.)

- [ ] **Step 3: Write `contracts/README.md` deploy runbook**

```markdown
# SkillPassCertificate — Foundry

## Setup
1. Install Foundry: `curl -L https://foundry.paradigm.xyz | bash && foundryup`
2. `cp .env.example .env` and fill in a **throwaway testnet** `PRIVATE_KEY`,
   `BASE_SEPOLIA_RPC_URL`, and `BASESCAN_API_KEY`. Never commit `.env`.
3. Fund the deployer with Base Sepolia ETH from a faucet.

## Test
    forge test -vvv

## Deploy to Base Sepolia (chainId 84532)
    source .env
    forge script script/Deploy.s.sol:Deploy \
      --rpc-url base_sepolia --broadcast --verify -vvv

## Export ABI (for the frontend in Phase 2)
    jq '.abi' out/SkillPassCertificate.sol/SkillPassCertificate.json \
      > ../deployments/SkillPassCertificate.abi.json
Record the deployed address in `../deployments/base-sepolia.json`.
```

- [ ] **Step 4: Create the deployments directory placeholder**

```bash
mkdir -p /Volumes/Project/skill-pass/deployments
touch /Volumes/Project/skill-pass/deployments/.gitkeep
```

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Project/skill-pass
git add contracts/script/Deploy.s.sol contracts/README.md deployments/.gitkeep
git commit -m "feat(contracts): add deploy script and runbook"
```

---

## Definition of Done (Phase 1)
- `forge test` is fully green (issue, access control, validation, soulbound, approval blocks, ERC-5192, tokenURI).
- `forge fmt` is clean.
- Contract deployed to Base Sepolia and verified on Basescan.
- ABI exported and deployed address recorded under `deployments/`.
- `.env` is git-ignored; no private key in history.

## Notes for the executor
- Quality gate (Solidity, not Sonar): `forge fmt` + `forge test` must pass; optionally run `slither .` and triage findings.
- Security lens (`senior-security`): owner-only minting, zero-address guard, bounded strings, no external calls before state writes (state is written before `_safeMint`).
- This plan is Phase 1 only. Phase 2 (frontend dApp) and Phase 3 (indexer + gateway) get their own plans.
