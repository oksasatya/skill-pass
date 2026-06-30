// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {SkillPassCertificate} from "../src/SkillPassCertificate.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

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

    function test_IssueCertificate_RevertWhen_NotOwner() public {
        address attacker = address(0xBAD);
        vm.prank(attacker);
        vm.expectRevert(abi.encodeWithSelector(Ownable.OwnableUnauthorizedAccount.selector, attacker));
        cert.issueCertificate(recipient, "X", "Y", "Z", "D", "ipfs://x");
    }

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
}
