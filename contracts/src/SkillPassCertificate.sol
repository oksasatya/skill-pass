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
