# Epoch-Based Rewards Distribution

This document provides an overview of the epoch-based rewards distribution system implemented in the Cosmos SDK distribution module.

## Overview

The distribution module has been modified to support epoch-based rewards distribution. Instead of distributing rewards at the end of every block, rewards are accumulated and distributed at the end of each epoch. This approach provides several benefits:

1. **Reduced computational overhead**: Processing rewards less frequently reduces the computational load on validators.
2. **Gas efficiency**: Fewer transactions are needed for reward distribution.
3. **Predictable distribution times**: Users know exactly when to expect their rewards.

## Implementation Details

The implementation takes a simple approach:

1. **Defer Distribution**: Instead of distributing rewards at the end of each block, the rewards are kept in the fee collector module account until an epoch ends.

2. **Epoch End Detection**: The `BeginBlocker` function checks if the current block is the end of an epoch:

   ```go
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

## Integration with Parent Codebase

To integrate with a parent codebase that manages epochs:

1. **Implement the EpochKeeper Interface**:

   ```go
   type EpochKeeper interface {
       // AllEpochInfos returns all the epoch infos
       AllEpochInfos(ctx sdk.Context) []EpochInfo
       // IsEpochEnd returns true if the current block is the end of an epoch
       IsEpochEnd(ctx sdk.Context, identifier string) bool
   }
   ```

2. **Provide Helper Functions**:

   ```go
   // Create a helper
   distributionHelper := distribution.CreateEpochBasedDistributionHelper(app.DistrKeeper)

   // In BeginBlock, always process the block to record the proposer
   distributionHelper.ProcessBlock(ctx)

   // At epoch end, distribute rewards
   if myEpochKeeper.IsEpochEnd(ctx, "day") {
       distributionHelper.ProcessEpochEnd(ctx)
   }
   ```

3. **Use SimpleEpochKeeper (Optional)**:

   ```go
   // Create a simple epoch keeper with daily and weekly epochs
   simpleEpochKeeper := distribution.NewSimpleEpochKeeper([]string{"day", "week"})

   // Set when epochs end
   if ctx.BlockHeight() % 10000 == 0 {
       simpleEpochKeeper.SetEpochEnd("day", true)
       defer simpleEpochKeeper.SetEpochEnd("day", false)
   }
   ```

## Benefits of This Approach

1. **Simplicity**: The implementation is straightforward and leverages existing code.
2. **Minimal Changes**: No need to modify the core reward distribution logic.
3. **Flexibility**: Works with any epoch definition from the parent codebase.
4. **No State Bloat**: No additional state is stored for tracking epoch rewards.

## Testing

The implementation includes tests to verify:

1. Rewards are not distributed until an epoch ends.
2. At epoch end, all accumulated rewards are distributed correctly.
3. Helper functions work as expected for integration with parent codebases.
