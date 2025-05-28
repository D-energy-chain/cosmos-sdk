# Dual Staking Rewards Implementation

This document outlines the implementation of a configurable dual staking rewards system in the Cosmos SDK distribution module, supporting both NFT stakers and native token stakers.

## Overview

The distribution module has been enhanced to support configurable reward distribution between NFT stakers and native token stakers. The system maintains the following allocation structure:

- **Community Tax**: 20% (configurable via `community_tax` parameter)
- **NFT Staking**: 56.25% of total rewards
- **Native Token Staking**: 18.75% of total rewards
- **Separate Pool**: 5% (configurable via `separate_pool_ratio` parameter)

## Parameters

### Parameters

1. **`community_tax`** (default: 0.20 = 20%)

   - Defines the fraction of total rewards allocated to the community pool
   - Default value results in 20% of total newly minted coins going to community pool

2. **`nft_staking_ratio`** (default: 0.75 = 75%)

   - Defines the fraction of staking rewards allocated to NFT stakers
   - This is a percentage of the staking rewards (after community tax and separate pool are deducted)
   - Default value results in 56.25% of total newly minted coins going to NFT stakers

3. **`native_staking_ratio`** (default: 0.25 = 25%)

   - Defines the fraction of staking rewards allocated to native token stakers
   - This is a percentage of the staking rewards (after community tax and separate pool are deducted)
   - Default value results in 18.75% of total newly minted coins going to native token stakers

4. **`separate_pool_ratio`** (default: 0.05 = 5%)
   - Defines the fraction of total rewards allocated to a separate pool
   - Default value results in 5% of total newly minted coins going to the separate pool

### Parameter Validation

The system enforces the following validation rules:

1. `community_tax` must be between 0 and 1.0
2. `nft_staking_ratio` must be between 0 and 1.0
3. `native_staking_ratio` must be between 0 and 1.0
4. `separate_pool_ratio` must be between 0 and 1.0
5. `nft_staking_ratio + native_staking_ratio` must equal exactly 1.0 (100% of staking rewards)
6. `community_tax + separate_pool_ratio` must be less than or equal to 1.0

## Implementation Details

### Protobuf Definition

The new parameters are defined in `proto/cosmos/distribution/v1beta1/distribution.proto`:

```protobuf
message Params {
  // ... existing fields ...

  string nft_staking_ratio = 5 [
    (cosmos_proto.scalar)  = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (amino.dont_omitempty) = true,
    (gogoproto.nullable)   = false
  ];

  string native_staking_ratio = 6 [
    (cosmos_proto.scalar)  = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (amino.dont_omitempty) = true,
    (gogoproto.nullable)   = false
  ];
}
```

### Keeper Methods

New getter methods have been added to the keeper:

```go
// GetNftStakingRatio returns the current NFT staking ratio parameter
func (k Keeper) GetNftStakingRatio(ctx context.Context) (math.LegacyDec, error)

// GetNativeStakingRatio returns the current native staking ratio parameter
func (k Keeper) GetNativeStakingRatio(ctx context.Context) (math.LegacyDec, error)
```

### Allocation Logic

The `AllocateTokensToValidator` function has been updated to:

1. Retrieve the configurable staking ratios from parameters
2. Apply these ratios to split rewards between NFT and native stakers
3. Handle edge cases where only one type of staking exists
4. Maintain existing commission and reward tracking logic

The allocation follows this logic:

```go
// If no NFT shares exist, all rewards go to native stakers
if nftShares.IsZero() {
    nativeRewards = tokens
} else if nativeShares.IsZero() {
    // All rewards go to NFT stakers
    nftRewards = tokens
} else {
    // Split based on configured ratios
    nftRewards = tokens.MulDecTruncate(nftStakingRatio)
    nativeRewards = tokens.MulDecTruncate(nativeStakingRatio)
}
```

## Integration with NFT Staking

The system is designed to work with the existing NFT staking infrastructure documented in `NFT_STAKING_REWARDS.md`.

### Prerequisites for Full Functionality

To enable the full dual staking rewards system, the following components need to be implemented:

1. **ValidatorI Interface Extension**: The `GetDelegatorNftShares()` method needs to be added to the staking module's `ValidatorI` interface
2. **NFT Delegation Tracking**: The staking module needs to track NFT delegations and shares
3. **NFT Staking Messages**: Transaction handlers for NFT delegation operations

### Current Implementation Status

The allocation logic is currently prepared for NFT staking but defaults to zero NFT shares until the staking module interface is updated. This means:

- The system currently works as a single native staking system
- Parameters are configurable and ready for dual staking
- Once NFT staking interface is implemented, the system will automatically support dual rewards

## Configuration Examples

### Default Configuration (Recommended)

```go
Params{
    CommunityTax:         math.LegacyNewDecWithPrec(20, 2), // 20%
    NftStakingRatio:      math.LegacyNewDecWithPrec(75, 2), // 75% of staking (60% total)
    NativeStakingRatio:   math.LegacyNewDecWithPrec(25, 2), // 25% of staking (20% total)
}
```

This results in:

- 20% to community pool
- 60% to NFT stakers (75% of 80% staking rewards)
- 20% to native token stakers (25% of 80% staking rewards)

### Alternative Configurations

**Equal Split Between Staking Types:**

```go
Params{
    CommunityTax:         math.LegacyNewDecWithPrec(20, 2), // 20%
    NftStakingRatio:      math.LegacyNewDecWithPrec(50, 2), // 50% of staking (40% total)
    NativeStakingRatio:   math.LegacyNewDecWithPrec(50, 2), // 50% of staking (40% total)
}
```

**Native Token Favored:**

```go
Params{
    CommunityTax:         math.LegacyNewDecWithPrec(10, 2), // 10%
    NftStakingRatio:      math.LegacyNewDecWithPrec(30, 2), // 30% of staking (27% total)
    NativeStakingRatio:   math.LegacyNewDecWithPrec(70, 2), // 70% of staking (63% total)
}
```

## Testing

The implementation includes comprehensive test coverage:

1. **Parameter Validation Tests**: Verify all validation rules are enforced
2. **Default Parameter Tests**: Ensure default values are valid
3. **Allocation Logic Tests**: Test reward distribution under various scenarios
4. **Edge Case Tests**: Handle cases with zero shares for either staking type

## Migration Considerations

When upgrading to this version:

1. **Parameter Migration**: Existing chains should carefully consider the new default `community_tax` value (20% vs previous 2%)
2. **Backward Compatibility**: The system is backward compatible for native staking only
3. **Testing**: Thoroughly test parameter changes on testnet before mainnet deployment

## Future Enhancements

1. **Dynamic Ratios**: Consider implementing time-based or stake-based dynamic ratio adjustments
2. **Multiple Asset Classes**: Extend support for additional asset types beyond NFTs
3. **Penalty Mechanisms**: Implement differential slashing or penalty mechanisms for different staking types

## Conclusion

This implementation provides a clean, configurable foundation for dual staking rewards that can accommodate various tokenomics requirements while maintaining the simplicity and reliability of the existing distribution system.
