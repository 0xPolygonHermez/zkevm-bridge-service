// SPDX-License-Identifier: AGPL-3.0

pragma solidity >=0.8.17;

import "./polygonZKEVMContracts/interfaces/IBridgeMessageReceiver.sol";
import "./polygonZKEVMContracts/interfaces/IPolygonZkEVMBridge.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * ZkEVMNFTBridge is an example contract to use the message layer of the PolygonZkEVMBridge to bridge NFTs
 */
contract PingReceiver is IBridgeMessageReceiver, Ownable {
    // Global Exit Root address
    IPolygonZkEVMBridge public immutable polygonZkEVMBridge;

    // Current network identifier
    uint32 public immutable networkID;

    // Address in the other network that will send the message
    address public pingSender;

    // Value sent from the other network
    uint256 public pingValue;

    /**
     * @param _polygonZkEVMBridge Polygon zkevm bridge address
     */
    constructor(IPolygonZkEVMBridge _polygonZkEVMBridge) {
        polygonZkEVMBridge = _polygonZkEVMBridge;
        networkID = polygonZkEVMBridge.networkID();
    }

    /**
     * @dev Emitted when a message is received from another network
     */
    event PingReceived(uint256 pingValue);

    /**
     * @dev Emitted when change the sender
     */
    event SetSender(address newPingSender);

    /**
     * @notice Set the sender of the message
     * @param newPingSender Address of the sender in the other network
     */
    function setSender(address newPingSender) external onlyOwner {
        pingSender = newPingSender;
        emit SetSender(newPingSender);
    }

    /**
     * @notice Verify merkle proof and withdraw tokens/ether
     * @param originAddress Origin address that the message was sended
     * @param originNetwork Origin network that the message was sended ( not usefull for this contract)
     * @param data Abi encoded metadata
     */
    function onMessageReceived(
        address originAddress,
        uint32 originNetwork,
        bytes memory data
    ) external payable override {
        // Can only be called by the bridge
        require(
            msg.sender == address(polygonZkEVMBridge),
            "PingReceiver::onMessageReceived: Not PolygonZkEVMBridge"
        );

        // Can only be called by the sender on the other network
        require(
            pingSender == originAddress,
            "PingReceiver::onMessageReceived: Not ping Sender"
        );

        // Decode data
        pingValue = abi.decode(data, (uint256));

        emit PingReceived(pingValue);
    }
}
