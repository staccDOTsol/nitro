// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

interface IXgasArbSys {
    function arbBlockNumber() external view returns (uint256);
}

/// @notice Xgas-specific enrollment registry for the EIP-8429 prototype.
/// @dev The execution client reads mapping slots 0 and 1 at block start.
///      No owner can enroll another address or shorten an exit.
contract ReferenceRegistryXgas {
    // Keep these mappings first and in this order: Nitro reads their slots.
    mapping(address => address) private _beneficiary; // slot 0
    mapping(address => uint256) private _exitBlock;   // slot 1

    IXgasArbSys private constant ARB_SYS = IXgasArbSys(0x0000000000000000000000000000000000000064);
    address public immutable sink;
    bytes32 public immutable vaultCodehash;
    uint256 public immutable windowBlocks;

    event Enrolled(address indexed account, address indexed beneficiary);
    event ExitScheduled(address indexed account, uint256 exitBlock);

    error AlreadyEnrolled();
    error ExitNotAvailable();
    error InvalidVault();
    error InvalidConfiguration();

    constructor(address sink_, bytes32 vaultCodehash_, uint256 windowBlocks_) {
        if (sink_ == address(0) || vaultCodehash_ == bytes32(0) || windowBlocks_ == 0) {
            revert InvalidConfiguration();
        }
        sink = sink_;
        vaultCodehash = vaultCodehash_;
        windowBlocks = windowBlocks_;
    }

    function enroll() external { _enroll(sink); }

    function enroll(address beneficiary) external {
        if (beneficiary.codehash != vaultCodehash) revert InvalidVault();
        _enroll(beneficiary);
    }

    function _enroll(address beneficiary) private {
        uint256 n = ARB_SYS.arbBlockNumber();
        address previous = _beneficiary[msg.sender];
        uint256 exitBlock = _exitBlock[msg.sender];
        if (previous != address(0) && (exitBlock == 0 || n < exitBlock + windowBlocks)) {
            revert AlreadyEnrolled();
        }
        _beneficiary[msg.sender] = beneficiary;
        _exitBlock[msg.sender] = 0;
        emit Enrolled(msg.sender, beneficiary);
    }

    function scheduleExit() external {
        uint256 n = ARB_SYS.arbBlockNumber();
        if (_beneficiary[msg.sender] == address(0) || _exitBlock[msg.sender] != 0) {
            revert ExitNotAvailable();
        }
        uint256 exitBlock = (n / windowBlocks + 2) * windowBlocks;
        _exitBlock[msg.sender] = exitBlock;
        emit ExitScheduled(msg.sender, exitBlock);
    }

    function isEnrolled(address account) external view returns (bool) {
        address beneficiary = _beneficiary[account];
        uint256 exitBlock = _exitBlock[account];
        return beneficiary != address(0) &&
            (exitBlock == 0 || ARB_SYS.arbBlockNumber() < exitBlock);
    }

    function beneficiaryOf(address account) external view returns (address) {
        return _beneficiary[account];
    }

    function exitBlockOf(address account) external view returns (uint256) {
        return _exitBlock[account];
    }
}
