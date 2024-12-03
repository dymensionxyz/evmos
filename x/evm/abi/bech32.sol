// SPDX-License-Identifier: MIT

pragma solidity >=0.7.0 <0.9.0;

interface IBech32CPC {
    /**
     * @dev Returns bech32-encoded value of the given 20 bytes address, using given HRP.
     * The second return value indicating whether the operation succeeded.
     */
    function bech32EncodeAddress(string memory hrp, address addr) external view returns (string memory, bool);

    /**
     * @dev Returns bech32-encoded value of the given 32 bytes address (ICA,...), using given HRP.
     * The second return value indicating whether the operation succeeded.
     */
    function bech32Encode32BytesAddress(string memory hrp, bytes32 addr) external view returns (string memory, bool);

    /**
     * @dev Returns bech32-encoded value of the given buffer, using given HRP.
     * Maximum allowed buffer size is 256 bytes.
     *
     * The second return value indicating whether the operation succeeded.
     */
    function bech32EncodeBytes(string memory hrp, bytes memory buffer) external view returns (string memory, bool);

    /**
     * @dev Decode given input bech32.
     * Maximum allowed input is 1023 characters.
     *
     * Returns:
     * - The first return value is the HRP.
     * - The second return value is the decoded buffer.
     * - The third return value indicating whether the operation succeeded.
     */
    function bech32Decode(string memory bech32) external view returns (string memory, bytes memory, bool);

    /**
     * @dev Returns the Bech32 prefix for account address.
     */
    function bech32AccountAddrPrefix() external view returns (string memory);

    /**
     * @dev Returns the Bech32 prefix for validator address.
     */
    function bech32ValidatorAddrPrefix() external view returns (string memory);

    /**
     * @dev Returns the Bech32 prefix for consensus node address.
     */
    function bech32ConsensusAddrPrefix() external view returns (string memory);

    /**
     * @dev Returns the Bech32 prefix for account public key.
     */
    function bech32AccountPubPrefix() external view returns (string memory);

    /**
     * @dev Returns the Bech32 prefix for validator public key.
     */
    function bech32ValidatorPubPrefix() external view returns (string memory);

    /**
     * @dev Returns the Bech32 prefix for consensus node public key.
     */
    function bech32ConsensusPubPrefix() external view returns (string memory);
}