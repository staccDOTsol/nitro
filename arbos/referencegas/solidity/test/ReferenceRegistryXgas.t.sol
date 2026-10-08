// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import {ReferenceRegistryXgas} from "../src/ReferenceRegistryXgas.sol";

interface VmReference {
    function mockCall(address callee, bytes calldata data, bytes calldata returnData) external;
}

contract ReferenceRegistryXgasTest {
    VmReference constant vm = VmReference(address(uint160(uint256(keccak256("hevm cheat code")))));
    address constant ARB_SYS = 0x0000000000000000000000000000000000000064;

    function _block(uint256 n) private {
        vm.mockCall(ARB_SYS, abi.encodeWithSignature("arbBlockNumber()"), abi.encode(n));
    }

    function testEnrollmentAndDelayedExitUseChildBlockNumber() public {
        _block(7);
        ReferenceRegistryXgas registry = new ReferenceRegistryXgas(address(0xBEEF), bytes32(uint256(1)), 10);
        registry.enroll();
        require(registry.isEnrolled(address(this)), "enrollment missing");
        require(registry.beneficiaryOf(address(this)) == address(0xBEEF), "wrong sink");
        registry.scheduleExit();
        require(registry.exitBlockOf(address(this)) == 20, "wrong exit boundary");

        _block(19);
        require(registry.isEnrolled(address(this)), "exit applied early");
        _block(20);
        require(!registry.isEnrolled(address(this)), "exit not applied");
        (bool early,) = address(registry).call(abi.encodeWithSignature("enroll()"));
        require(!early, "re-enrollment allowed inside cooldown");

        _block(30);
        registry.enroll();
        require(registry.isEnrolled(address(this)), "re-enrollment failed");
        require(registry.exitBlockOf(address(this)) == 0, "exit not cleared");
    }
}
