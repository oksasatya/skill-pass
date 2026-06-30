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
