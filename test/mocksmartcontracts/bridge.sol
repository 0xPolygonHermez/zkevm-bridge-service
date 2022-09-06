// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.15;

import "./lib/DepositContract.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import "./lib/TokenWrapped.sol";
import "./interfaces/IGlobalExitRootManager.sol";
import "@openzeppelin/contracts-upgradeable/proxy/ClonesUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/IERC20MetadataUpgradeable.sol";

/**
 * Bridge that will be deployed on both networks Ethereum and Polygon zkEVM
 * Contract responsible to manage the token interactions with other networks
 */
contract Bridge is DepositContract {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    // Wrapped Token information struct
    struct TokenInformation {
        uint32 originNetwork;
        address originTokenAddress;
    }

    // bytes4(keccak256(bytes("permit(address,address,uint256,uint256,uint8,bytes32,bytes32)")));
    bytes4 constant _PERMIT_SIGNATURE = 0xd505accf;

    // Mainnet indentifier
    uint32 public constant MAINNET_NETWORK_ID = 0;

    // Network identifier
    uint32 public networkID;

    // Leaf index --> claimed
    mapping(uint256 => bool) public claimNullifier;

    // keccak256(OriginNetwork || tokenAddress) --> Wrapped token address
    mapping(bytes32 => address) public tokenInfoToWrappedToken;

    // Wrapped token Address --> Origin token information
    mapping(address => TokenInformation) public wrappedTokenToTokenInfo;

    // Global Exit Root address
    IGlobalExitRootManager public globalExitRootManager;

    // Addres of the token wrapped implementation
    address public tokenImplementation;

    /**
     * @param _networkID networkID
     * @param _globalExitRootManager global exit root manager address
     */
    function initialize(
        uint32 _networkID,
        IGlobalExitRootManager _globalExitRootManager
    ) public initializer {
        networkID = _networkID;
        globalExitRootManager = _globalExitRootManager;
        tokenImplementation = address(new TokenWrapped());
        __DepositContract_init();
    }

    /**
     * @dev Emitted when a bridge some tokens to another network
     */
    event BridgeEvent(
        uint32 originNetwork,
        address originTokenAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes metadata,
        uint32 depositCount
    );

    /**
     * @dev Emitted when a claim is done from another network
     */
    event ClaimEvent(
        uint32 index,
        uint32 originNetwork,
        address originTokenAddress,
        address destinationAddress,
        uint256 amount
    );

    /**
     * @dev Emitted when a a new wrapped token is created
     */
    event NewWrappedToken(
        uint32 originNetwork,
        address originTokenAddress,
        address wrappedTokenAddress
    );

    /**
     * @notice Deposit add a new leaf to the merkle tree
     * @param token Token address, 0 address is reserved for ether
     * @param destinationNetwork Network destination
     * @param destinationAddress Address destination
     * @param amount Amount of tokens
     * @param permitData Raw data of the call `permit` of the token
     */
    function bridge(
        address token,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes calldata permitData
    ) public payable {
        require(
            destinationNetwork != networkID,
            "Bridge::bridge: DESTINATION_CANT_BE_ITSELF"
        );

        address originTokenAddress;
        uint32 originNetwork;
        bytes memory metadata;

        if (token == address(0)) {
            // Ether transfer
            require(
                msg.value == amount,
                "Bridge::bridge: AMOUNT_DOES_NOT_MATCH_MSG_VALUE"
            );

            // Ether is treated as ether from mainnet
            originNetwork = MAINNET_NETWORK_ID;
        } else {
            TokenInformation memory tokenInfo = wrappedTokenToTokenInfo[token];
            originTokenAddress = token;
            originNetwork = networkID;

            // Encode metadata
            metadata = abi.encode(
                IERC20MetadataUpgradeable(token).name(),
                IERC20MetadataUpgradeable(token).symbol(),
                IERC20MetadataUpgradeable(token).decimals()
            );
            
        }

        emit BridgeEvent(
            originNetwork,
            originTokenAddress,
            destinationNetwork,
            destinationAddress,
            amount,
            metadata,
            uint32(depositCount)
        );

        // Update the new exit root to the exit root manager
        globalExitRootManager.updateExitRoot(getDepositRoot());
    }

    /**
     * @notice Verify merkle proof and withdraw tokens/ether
     * @param smtProof Smt proof
     * @param index Index of the leaf
     * @param mainnetExitRoot Mainnet exit root
     * @param rollupExitRoot Rollup exit root
     * @param originNetwork Origin network
     * @param originTokenAddress  Origin token address, 0 address is reserved for ether
     * @param destinationNetwork Network destination, must be 0 ( mainnet)
     * @param destinationAddress Address destination
     * @param amount Amount of tokens
     * @param metadata abi encoded metadata if any, empty otherwise
     */
    function claim(
        bytes32[] memory smtProof,
        uint32 index,
        bytes32 mainnetExitRoot,
        bytes32 rollupExitRoot,
        uint32 originNetwork,
        address originTokenAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes memory metadata
    ) public {
        // Check nullifier
        require(
            claimNullifier[index] == false,
            "Bridge::claim: ALREADY_CLAIMED"
        );

        // Update nullifier
        claimNullifier[index] = true;

        // Transfer funds
        if (originTokenAddress == address(0)) {

        } else {
            // Create a new wrapped erc20
            emit NewWrappedToken(
                originNetwork,
                originTokenAddress,
                address(0)
            );
        }

        emit ClaimEvent(
            index,
            originNetwork,
            originTokenAddress,
            destinationAddress,
            amount
        );
    }

    /**
     * @notice Returns the precalculated address of a wrapper using the token information
     * @param originNetwork Origin network
     * @param originTokenAddress Origin token address, 0 address is reserved for ether
     */
    function precalculatedWrapperAddress(
        uint32 originNetwork,
        address originTokenAddress
    ) public view returns (address) {
        bytes32 salt = keccak256(
            abi.encodePacked(originNetwork, originTokenAddress)
        );
        return
            ClonesUpgradeable.predictDeterministicAddress(
                tokenImplementation,
                salt
            );
    }

    /**
     * @notice Returns the address of a wrapper using the token information if already exist
     * @param originNetwork Origin network
     * @param originTokenAddress Origin token address, 0 address is reserved for ether
     */
    function getTokenWrappedAddress(
        uint32 originNetwork,
        address originTokenAddress
    ) public view returns (address) {
        return
            tokenInfoToWrappedToken[
                keccak256(abi.encodePacked(originNetwork, originTokenAddress))
            ];
    }

    /**
     * @notice Function to extract the selector of a bytes calldata
     * @param _data The calldata bytes
     */
    function _getSelector(bytes memory _data)
        private
        pure
        returns (bytes4 sig)
    {
        assembly {
            sig := mload(add(_data, 32))
        }
    }

    /**
     * @notice Function to call token permit method of extended ERC20
     + @param token ERC20 token address
     * @param amount Quantity that is expected to be allowed
     * @param permitData Raw data of the call `permit` of the token
     */
    function _permit(
        address token,
        uint256 amount,
        bytes calldata permitData
    ) internal {
        bytes4 sig = _getSelector(permitData);
        require(
            sig == _PERMIT_SIGNATURE,
            "Bridge::_permit: NOT_VALID_CALL"
        );
        (
            address owner,
            address spender,
            uint256 value,
            uint256 deadline,
            uint8 v,
            bytes32 r,
            bytes32 s
        ) = abi.decode(
                permitData[4:],
                (address, address, uint256, uint256, uint8, bytes32, bytes32)
            );
        require(
            owner == msg.sender,
            "Bridge::_permit: PERMIT_OWNER_MUST_BE_THE_SENDER"
        );
        require(
            spender == address(this),
            "Bridge::_permit: SPENDER_MUST_BE_THIS"
        );
        require(
            value == amount,
            "Bridge::_permit: PERMIT_AMOUNT_DOES_NOT_MATCH"
        );

        // we call without checking the result, in case it fails and he doesn't have enough balance
        // the following transferFrom should be fail. This prevents DoS attacks from using a signature
        // before the smartcontract call
        /* solhint-disable avoid-low-level-calls */
        address(token).call(
            abi.encodeWithSelector(
                _PERMIT_SIGNATURE,
                owner,
                spender,
                value,
                deadline,
                v,
                r,
                s
            )
        );
    }
}