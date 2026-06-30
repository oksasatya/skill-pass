// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC721} from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

/// @title SkillPassCertificate
/// @notice Soulbound (non-transferable) ERC-721 certificate. Owner-only issuance.
contract SkillPassCertificate is ERC721, Ownable {
    struct Certificate {
        string title;
        string recipientName;
        string issuerName;
        string description;
        string metadataURI;
        uint256 issuedAt;
    }

    error ZeroRecipient();
    error StringTooLong();
    error Soulbound();
    error ApprovalDisabled();

    uint256 private constant MAX_TITLE = 200;
    uint256 private constant MAX_NAME = 100;
    uint256 private constant MAX_DESC = 1000;
    uint256 private constant MAX_URI = 300;

    event CertificateIssued(
        uint256 indexed tokenId,
        address indexed recipient,
        string title,
        string issuerName,
        uint256 issuedAt
    );

    uint256 private _nextTokenId = 1;
    mapping(uint256 tokenId => Certificate) private _certificates;

    constructor(address initialOwner)
        ERC721("SkillPass Certificate", "SKILL")
        Ownable(initialOwner)
    {}

    function issueCertificate(
        address recipient,
        string calldata title,
        string calldata recipientName,
        string calldata issuerName,
        string calldata description,
        string calldata metadataURI
    ) external onlyOwner returns (uint256 tokenId) {
        if (recipient == address(0)) revert ZeroRecipient();
        if (bytes(title).length > MAX_TITLE) revert StringTooLong();
        if (bytes(recipientName).length > MAX_NAME) revert StringTooLong();
        if (bytes(issuerName).length > MAX_NAME) revert StringTooLong();
        if (bytes(description).length > MAX_DESC) revert StringTooLong();
        if (bytes(metadataURI).length > MAX_URI) revert StringTooLong();
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
}
