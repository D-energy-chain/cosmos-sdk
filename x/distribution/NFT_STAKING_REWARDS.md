# NFT Staking Rewards Implementation

This document outlines the implementation of NFT staking rewards in the Cosmos SDK distribution module.

## Overview

The distribution module has been modified to support NFT staking rewards. The distribution of newly minted tokens is as follows:

- 20% goes to the community pool
- 60% goes to NFT stakers
- 20% goes to native token stakers

## Implementation Details

### 1. Validator Interface Extension

The `ValidatorI` interface in the staking module has been extended to include a new method:

```go
GetDelegatorNftShares() math.LegacyDec // total NFT delegator shares
```

This method returns the total NFT delegator shares associated with a validator.

### 2. Validator Struct Extension

The `Validator` struct in the staking module has been extended to include NFT delegation-related fields:

```go
// NFT delegation related fields
TotalNftDelegation      math.Int            // total NFT delegation
DelegatorNftShares      math.LegacyDec      // total NFT delegator shares
NftDelegations          []*NFTDelegation    // list of NFT delegations
MinNftSelfDelegation    math.Int            // minimum NFT self delegation
NftUnbondingIds         []uint64            // list of NFT unbonding IDs
```

And a new method has been added to retrieve the NFT delegator shares:

```go
func (v Validator) GetDelegatorNftShares() math.LegacyDec { return v.DelegatorNftShares }
```

### 3. NFT Delegation Struct

A new `NFTDelegation` struct has been added to represent NFT delegations:

```go
type NFTDelegation struct {
    DelegatorAddress    string           // delegator address
    ValidatorAddress    string           // validator address
    NftContractAddress  string           // NFT contract address
    TokenId             uint64           // token ID
    Shares              math.LegacyDec   // shares
}
```

### 4. Distribution Module Modification

The `AllocateTokensToValidator` function in the distribution module has been modified to split rewards between NFT stakers and native token stakers:

```go
func (k Keeper) AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error {
    // Get the validator's NFT and native token shares
    nftShares := val.GetDelegatorNftShares()
    nativeShares := val.GetDelegatorShares()

    // If there are no NFT shares, all rewards go to native token stakers
    if nftShares.IsZero() {
        // Original logic: split between validator commission and delegators
        commission := tokens.MulDec(val.GetCommission())
        shared := tokens.Sub(commission)
        // Update rewards...
    } else if nativeShares.IsZero() {
        // All rewards go to NFT stakers
        commission := tokens.MulDec(val.GetCommission())
        shared := tokens.Sub(commission)
        // Update rewards...
    } else {
        // Split rewards between NFT stakers (75%) and native token stakers (25%)
        nftStakingRatio := math.LegacyNewDecWithPrec(75, 2)    // 75%
        nativeStakingRatio := math.LegacyNewDecWithPrec(25, 2) // 25%

        // Calculate rewards for each type
        nftRewards := tokens.MulDecTruncate(nftStakingRatio)
        nativeRewards := tokens.MulDecTruncate(nativeStakingRatio)

        // Apply commission to each reward type
        nftCommission := nftRewards.MulDec(val.GetCommission())
        nftShared := nftRewards.Sub(nftCommission)

        nativeCommission := nativeRewards.MulDec(val.GetCommission())
        nativeShared := nativeRewards.Sub(nativeCommission)

        // Combine commissions and update rewards...
    }

    // Update validator accumulated commission and rewards...
}
```

This approach preserves the existing distribution logic while adding support for NFT staking rewards. The key points are:

1. The validator power calculation remains unchanged, which means the `AllocateTokens` function continues to work as before.
2. When tokens are allocated to a validator, they are split between NFT stakers and native token stakers.
3. The validator receives commission on both types of rewards.
4. The rewards are tracked in the existing reward structures.

## Epoch-Based Reward Distribution

The distribution module now exclusively uses epoch-based reward distribution instead of per-block distribution. This provides several benefits:

1. **Reduced computational overhead**: Processing rewards less frequently reduces the computational load on validators.
2. **Gas efficiency**: Fewer transactions are needed for reward distribution.
3. **Predictable distribution times**: Users know exactly when to expect their rewards.

### Implementation with Parent Codebase's Epoch Module

The distribution module integrates with the parent codebase's epoch module to enable epoch-based reward distribution. The parent codebase must implement the `EpochKeeper` interface defined in the `x/distribution/exported` package:

```go
// EpochKeeper defines the expected epoch keeper interface
type EpochKeeper interface {
    // AllEpochInfos returns all the epoch infos
    AllEpochInfos(ctx sdk.Context) []EpochInfo
    // IsEpochEnd returns true if the current block is the end of an epoch
    IsEpochEnd(ctx sdk.Context, identifier string) bool
}

// EpochInfo defines the epoch info structure
type EpochInfo struct {
    Identifier string
}
```

### Simple Approach to Epoch-Based Distribution

The implementation takes a simple approach:

1. **Defer Distribution**: Instead of distributing rewards at the end of each block, the rewards are kept in the fee collector module account until an epoch ends.

2. **Epoch End Detection**: The `BeginBlocker` function checks if the current block is the end of an epoch:

```go
// In distribution module's BeginBlocker
func BeginBlocker(ctx sdk.Context, k keeper.Keeper, ek exported.EpochKeeper) error {
    // Record proposer and other setup

    // Check if any epoch has ended
    epochs := ek.AllEpochInfos(ctx)
    for _, epoch := range epochs {
        if ek.IsEpochEnd(ctx, epoch.Identifier) {
            // At epoch end, distribute all accumulated rewards
            if err := k.AllocateTokens(ctx, previousTotalPower, ctx.VoteInfos()); err != nil {
                return err
            }
            return nil
        }
    }

    // If not at epoch end, just collect fees but don't distribute them
    // The fees will remain in the fee collector module account until the epoch ends
    return nil
}
```

3. **Reward Distribution**: At the end of an epoch, the standard `AllocateTokens` function is called, which:
   - Collects all accumulated fees from the fee collector module account
   - Distributes them according to the standard distribution rules, including:
     - Community pool allocation
     - Validator commission
     - Delegator rewards (both for NFT and native token stakers)

This approach maintains the existing reward calculation logic while simply deferring the actual distribution to epoch boundaries, making it both simple to implement and easy to understand.

## Reward Distribution

### For Validators

Validators receive commission on both NFT staking rewards and native token staking rewards. The commission rate is the same for both types of rewards.

### For Delegators

Delegator rewards are calculated using the F1 distribution mechanism, which is the same mechanism used for native token staking rewards. When a delegator withdraws their rewards:

1. For native token delegations, the rewards are calculated based on the delegator's share of the validator's total native token shares.
2. For NFT delegations, the rewards are calculated based on the delegator's share of the validator's total NFT shares.

### For Self-Delegations

Validators receive rewards for their self-delegations just like any other delegator. If a validator has both native token self-delegations and NFT self-delegations, they receive rewards for both.

## Usage

To use NFT staking rewards in your chain:

1. Ensure that the staking module includes the NFT delegation functionality
2. Implement the NFT delegation messages and handlers that update the `DelegatorNftShares` field in the validator struct
3. Update the distribution module to use the modified `AllocateTokensToValidator` function
4. Configure the epochs module with appropriate epoch duration (e.g., daily, weekly)
5. Update the distribution module to use epoch-based reward distribution

## Testing

Test cases have been added to verify the NFT staking rewards distribution:

```go
func TestAllocateTokensToValidatorWithNFTAndNative(t *testing.T) {
    // Tests allocation with both native token and NFT delegations
}

func TestAllocateTokensToValidatorWithOnlyNFT(t *testing.T) {
    // Tests allocation with only NFT delegations
}

func TestEpochBasedRewardDistribution(t *testing.T) {
    // Tests the accumulation and distribution of rewards at epoch boundaries
}
```

These tests verify that:

- When a validator has both native token and NFT delegations, rewards are split 75/25
- When a validator has only NFT delegations, all rewards go to NFT stakers
- When a validator has only native token delegations, all rewards go to native token stakers
- The validator receives the correct commission on all rewards
- Rewards are properly accumulated during an epoch and distributed at the epoch boundary

## Integration with Evmos

For Evmos chains, you'll need to:

1. Implement the NFT staking module that handles ERC-1155 tokens
2. Update the staking module to track NFT delegations and update the `DelegatorNftShares` field
3. Ensure the inflation module directs 80% of newly minted tokens to the distribution module
4. Use the modified distribution module to split rewards between NFT stakers and native token stakers
5. Configure the epochs module with appropriate epoch duration for your chain

## Integration with Parent Codebase

Since the epochs are managed by the parent codebase, you'll need to ensure proper integration:

1. **Implement the EpochKeeper Interface**:
   The parent codebase must implement the `exported.EpochKeeper` interface:

   ```go
   // In parent codebase
   import (
       "github.com/cosmos/cosmos-sdk/types"
       "github.com/cosmos/cosmos-sdk/x/distribution/exported"
   )

   type MyEpochKeeper struct {
       // Your implementation
   }

   // Implement the EpochKeeper interface
   func (k MyEpochKeeper) AllEpochInfos(ctx sdk.Context) []exported.EpochInfo {
       // Return all epochs from your implementation
       var epochs []exported.EpochInfo
       for _, epoch := range k.GetAllEpochs(ctx) {
           epochs = append(epochs, exported.EpochInfo{
               Identifier: epoch.Identifier,
           })
       }
       return epochs
   }

   func (k MyEpochKeeper) IsEpochEnd(ctx sdk.Context, identifier string) bool {
       // Check if the current block is the end of the specified epoch
       return k.IsCurrentBlockEpochEnd(ctx, identifier)
   }
   ```

2. **Provide the EpochKeeper to the Distribution Module**:
   When setting up the app, provide your implementation of the EpochKeeper to the distribution module:

   ```go
   // In app.go or equivalent
   app.DistrKeeper = distrkeeper.NewKeeper(
       // Other parameters...
   )

   // Create distribution module with epoch keeper
   app.DistrModule = distribution.NewAppModule(
       app.AppCodec,
       app.DistrKeeper,
       app.AccountKeeper,
       app.BankKeeper,
       app.StakingKeeper,
       app.MyEpochKeeper, // Your epoch keeper implementation
       app.GetSubspace(distribution.ModuleName),
   )
   ```

3. **Use Helper Functions (Optional)**:
   The distribution module provides helper functions to simplify integration:

   ```go
   // Create a helper
   distributionHelper := distribution.CreateEpochBasedDistributionHelper(app.DistrKeeper)

   // In BeginBlock, always process the block to record the proposer
   app.BeginBlocker = func(ctx sdk.Context) {
       // Other BeginBlock operations...

       // Record the proposer for this block
       distributionHelper.ProcessBlock(ctx)

       // Check if any epoch has ended
       if myEpochKeeper.IsEpochEnd(ctx, "day") {
           // Process epoch end and distribute rewards
           distributionHelper.ProcessEpochEnd(ctx)
       }
   }
   ```

4. **Use SimpleEpochKeeper (Optional)**:
   If you don't have your own epoch implementation, you can use the provided SimpleEpochKeeper:

   ```go
   // Create a simple epoch keeper with daily and weekly epochs
   simpleEpochKeeper := distribution.NewSimpleEpochKeeper([]string{"day", "week"})

   // In your epoch tracking logic, set when epochs end
   app.EndBlocker = func(ctx sdk.Context) {
       // Check if this is the end of a day (e.g., every 10000 blocks)
       if ctx.BlockHeight() % 10000 == 0 {
           simpleEpochKeeper.SetEpochEnd("day", true)
           defer simpleEpochKeeper.SetEpochEnd("day", false)
       }

       // Check if this is the end of a week (e.g., every 70000 blocks)
       if ctx.BlockHeight() % 70000 == 0 {
           simpleEpochKeeper.SetEpochEnd("week", true)
           defer simpleEpochKeeper.SetEpochEnd("week", false)
       }

       // Other EndBlock operations...
   }
   ```

5. **Testing**:
   Ensure your implementation correctly triggers reward distribution at epoch boundaries by writing integration tests that simulate epoch transitions.

This simplified approach doesn't require any special initialization or state management for epochs, as it uses the existing reward distribution mechanism and just defers it to epoch boundaries.
