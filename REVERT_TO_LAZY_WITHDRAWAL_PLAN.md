# Revert to Lazy Withdrawal - Implementation Plan

## 📋 Overview

Revert from automatic distribution to lazy (manual) withdrawal mechanism while preserving all the calculation improvements, bug fixes, and logging enhancements made during automatic distribution implementation.

**Rationale**: Automatic distribution to all delegators every epoch is computationally expensive and consumes significant gas. Lazy withdrawal allows delegators to claim rewards on-demand, distributing the gas cost over time and to individual users.

---

## 🎯 Goals

1. **Remove Automatic Distribution**: Stop distributing at epoch end
2. **Restore Manual Withdrawal**: Bring back transaction endpoints for claiming rewards
3. **Keep Improvements**: Maintain all bug fixes, calculations, and logging
4. **Keep Efficiency**: Maintain O(1) reward calculation using periods and cumulative ratios

---

## ✅ What to Keep (Improvements)

### Calculation Logic:
- ✅ Fixed period tracking (`resetDelegatorStartingInfoToPeriod`)
- ✅ Correct ending period usage (`currentRewards.Period - 1`)
- ✅ NFT + Native reward calculation
- ✅ Rounding/intersection logic for outstanding rewards
- ✅ Pro-rating support (when implemented)

### Logging:
- ✅ Separate NFT/Native reward breakdown
- ✅ Period increment logs
- ✅ Historical rewards storage logs
- ✅ Debug logs for reward calculation

### Data Structures:
- ✅ `DelegatorStartingInfo` with Height
- ✅ `ValidatorHistoricalRewards` with cumulative ratios
- ✅ Period-based F1 algorithm
- ✅ NFT stake tracking

---

## ❌ What to Remove

### Automatic Distribution:
- ❌ `DistributeRewardsToAllDelegators` function
- ❌ Call to automatic distribution in `AfterEpochEnd` hook
- ❌ `distributeRewardsToSingleDelegator` function (or repurpose for manual withdrawal)
- ❌ Distribution metrics and logging specific to automatic distribution
- ❌ `min_auto_distribution_amount` parameter (not needed for lazy withdrawal)

### Hook Behavior:
- ❌ Distribution on `BeforeDelegationRemoved`
- ❌ Distribution on `BeforeNFTDelegationRemoved`
- ❌ Period skip checks in removal hooks

---

## 🔄 What to Restore

### Proto Definitions:
```protobuf
// Restore withdrawal message
message MsgWithdrawDelegatorReward {
  option (cosmos.msg.v1.signer) = "delegator_address";
  
  string delegator_address = 1;
  string validator_address = 2;
}

message MsgWithdrawDelegatorRewardResponse {
  repeated cosmos.base.v1beta1.Coin amount = 1;
}

// Add back to service
service Msg {
  // ... existing methods ...
  
  rpc WithdrawDelegatorReward(MsgWithdrawDelegatorReward) 
    returns (MsgWithdrawDelegatorRewardResponse);
}
```

### Message Handler:
```go
// x/distribution/keeper/msg_server.go
func (k msgServer) WithdrawDelegatorReward(
    ctx context.Context, 
    msg *types.MsgWithdrawDelegatorReward,
) (*types.MsgWithdrawDelegatorRewardResponse, error) {
    // Use the improved calculation logic
    // Withdraw rewards for a single delegator-validator pair
}
```

### CLI Commands:
```go
// x/distribution/client/cli/tx.go
func NewWithdrawRewardsCmd() *cobra.Command {
    // Command to withdraw from single validator
}

func NewWithdrawAllRewardsCmd() *cobra.Command {
    // Command to withdraw from all validators
}
```

---

## 📝 Implementation Steps

### **Phase 1: Remove Automatic Distribution**

#### Step 1.1: Remove Auto-Distribution Function
- **File**: `x/distribution/keeper/auto_distribution.go`
- **Action**: DELETE entire file or comment out `DistributeRewardsToAllDelegators`
- **Keep**: The calculation logic can be reused for manual withdrawal

#### Step 1.2: Remove Hook Call
- **File**: `x/distribution/keeper/hooks.go`
- **Function**: `AfterEpochEnd`
- **Action**: Remove call to `DistributeRewardsToAllDelegators`

```go
func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
    // ... existing allocation logic ...
    
    // Increment periods for all validators
    if err := h.k.IncrementAllValidatorPeriods(ctx); err != nil {
        ctx.Logger().Error("Failed to increment validator periods", "error", err)
        return
    }
    
    // ❌ REMOVE THIS:
    // if err := h.k.DistributeRewardsToAllDelegators(ctx); err != nil {
    //     ctx.Logger().Error("Failed to automatically distribute", "error", err)
    // }
    
    // ✅ Rewards now accumulate and wait for manual withdrawal
    ctx.Logger().Info("Epoch rewards allocated - awaiting manual withdrawal")
}
```

#### Step 1.3: Simplify Removal Hooks
- **File**: `x/distribution/keeper/hooks.go`
- **Functions**: `BeforeDelegationRemoved`, `BeforeNFTDelegationRemoved`
- **Action**: Restore original behavior - withdraw before removal

```go
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
    // Check if delegation exists
    del, err := h.k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
    if err != nil || del == nil {
        return nil
    }
    
    // Withdraw accumulated rewards before removing delegation
    if _, err := h.k.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
        h.k.Logger(ctx).Error("Failed to withdraw rewards before delegation removal", "error", err)
    }
    
    return nil
}
```

#### Step 1.4: Remove Distribution Parameter
- **File**: `proto/cosmos/distribution/v1beta1/distribution.proto`
- **Action**: Remove `min_auto_distribution_amount` field
- **Note**: This is a state-breaking change, consider leaving it unused if migration is difficult

---

### **Phase 2: Restore Manual Withdrawal**

#### Step 2.1: Restore Proto Definitions
- **File**: `proto/cosmos/distribution/v1beta1/tx.proto`
- **Action**: Add back `MsgWithdrawDelegatorReward` and response
- **Action**: Add back RPC endpoint in Msg service

#### Step 2.2: Regenerate Proto Files
```bash
make proto-gen
```

#### Step 2.3: Implement Message Handler
- **File**: `x/distribution/keeper/msg_server.go`
- **Action**: Implement `WithdrawDelegatorReward` using improved calculation logic

```go
func (k msgServer) WithdrawDelegatorReward(
    ctx context.Context,
    msg *types.MsgWithdrawDelegatorReward,
) (*types.MsgWithdrawDelegatorRewardResponse, error) {
    valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(msg.ValidatorAddress)
    if err != nil {
        return nil, err
    }
    
    delAddr, err := k.authKeeper.AddressCodec().StringToBytes(msg.DelegatorAddress)
    if err != nil {
        return nil, err
    }
    
    // Get validator
    val, err := k.stakingKeeper.Validator(ctx, valAddr)
    if err != nil {
        return nil, err
    }
    
    if val == nil {
        return nil, types.ErrNoValidatorExists
    }
    
    // Check if delegator starting info exists
    hasInfo, err := k.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
    if err != nil || !hasInfo {
        // No rewards to withdraw
        return &types.MsgWithdrawDelegatorRewardResponse{
            Amount: sdk.NewCoins(),
        }, nil
    }
    
    // Get current period (use period - 1 as ending period)
    currentRewards, err := k.GetValidatorCurrentRewards(ctx, valAddr)
    if err != nil {
        return nil, err
    }
    endingPeriod := currentRewards.Period - 1
    
    // Get starting info
    startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
    if err != nil {
        return nil, err
    }
    
    // Calculate rewards using improved logic
    var totalRewards sdk.DecCoins
    
    // Native rewards
    if !startingInfo.Stake.IsZero() {
        nativeRewards, err := k.calculateDelegationRewardsBetween(
            ctx, val,
            startingInfo.PreviousPeriod,
            endingPeriod,
            startingInfo.Stake,
        )
        if err != nil {
            return nil, err
        }
        totalRewards = totalRewards.Add(nativeRewards...)
    }
    
    // NFT rewards
    if !startingInfo.NftStake.IsZero() {
        nftRewards, err := k.calculateNFTDelegationRewardsBetween(
            ctx, val,
            startingInfo.PreviousPeriod,
            endingPeriod,
            startingInfo.NftStake,
        )
        if err != nil {
            return nil, err
        }
        totalRewards = totalRewards.Add(nftRewards...)
    }
    
    // If no rewards, return early
    if totalRewards.IsZero() {
        return &types.MsgWithdrawDelegatorRewardResponse{
            Amount: sdk.NewCoins(),
        }, nil
    }
    
    // Apply intersection with outstanding rewards
    outstanding, err := k.GetValidatorOutstandingRewardsCoins(ctx, valAddr)
    if err != nil {
        return nil, err
    }
    totalRewards = totalRewards.Intersect(outstanding)
    
    // Truncate to coins
    coins, _ := totalRewards.TruncateDecimal()
    
    // Get withdraw address
    withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAddr)
    if err != nil {
        return nil, err
    }
    
    // Transfer rewards
    err = k.bankKeeper.SendCoinsFromModuleToAccount(
        ctx, types.ModuleName, withdrawAddr, coins,
    )
    if err != nil {
        return nil, err
    }
    
    // Update outstanding rewards
    outstanding = outstanding.Sub(totalRewards)
    err = k.SetValidatorOutstandingRewards(ctx, valAddr, types.ValidatorOutstandingRewards{
        Rewards: outstanding,
    })
    if err != nil {
        return nil, err
    }
    
    // Reset delegator starting info to current ending period
    err = k.resetDelegatorStartingInfoToPeriod(ctx, valAddr, delAddr, endingPeriod)
    if err != nil {
        return nil, err
    }
    
    // Emit event
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    sdkCtx.EventManager().EmitEvent(
        sdk.NewEvent(
            types.EventTypeWithdrawRewards,
            sdk.NewAttribute(sdk.AttributeKeyAmount, coins.String()),
            sdk.NewAttribute(types.AttributeKeyValidator, msg.ValidatorAddress),
        ),
    )
    
    // Log with breakdown
    k.Logger(ctx).Info("💰 Rewards withdrawn",
        "delegator", msg.DelegatorAddress,
        "validator", msg.ValidatorAddress,
        "amount", coins.String(),
    )
    
    return &types.MsgWithdrawDelegatorRewardResponse{
        Amount: coins,
    }, nil
}
```

#### Step 2.4: Implement CLI Commands
- **File**: `x/distribution/client/cli/tx.go`
- **Action**: Restore withdrawal commands

```go
func NewWithdrawRewardsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "withdraw-rewards [validator-addr]",
        Short: "Withdraw rewards from a given validator address",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            clientCtx, err := client.GetClientTxContext(cmd)
            if err != nil {
                return err
            }
            
            delAddr := clientCtx.GetFromAddress()
            valAddr := args[0]
            
            msg := types.NewMsgWithdrawDelegatorReward(
                delAddr.String(), valAddr,
            )
            
            return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
        },
    }
    
    flags.AddTxFlagsToCmd(cmd)
    return cmd
}

func NewWithdrawAllRewardsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "withdraw-all-rewards",
        Short: "Withdraw all delegations rewards for a delegator",
        RunE: func(cmd *cobra.Command, args []string) error {
            clientCtx, err := client.GetClientTxContext(cmd)
            if err != nil {
                return err
            }
            
            delAddr := clientCtx.GetFromAddress()
            
            // Query all validators this delegator has delegated to
            queryClient := types.NewQueryClient(clientCtx)
            res, err := queryClient.DelegatorValidators(
                cmd.Context(),
                &types.QueryDelegatorValidatorsRequest{
                    DelegatorAddress: delAddr.String(),
                },
            )
            if err != nil {
                return err
            }
            
            // Create withdrawal message for each validator
            msgs := make([]sdk.Msg, 0, len(res.Validators))
            for _, valAddr := range res.Validators {
                msgs = append(msgs, types.NewMsgWithdrawDelegatorReward(
                    delAddr.String(), valAddr,
                ))
            }
            
            return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
        },
    }
    
    flags.AddTxFlagsToCmd(cmd)
    return cmd
}
```

#### Step 2.5: Register CLI Commands
- **File**: `x/distribution/client/cli/tx.go`
- **Function**: `NewTxCmd`
- **Action**: Add withdrawal commands back

```go
func NewTxCmd() *cobra.Command {
    // ... existing setup ...
    
    distTxCmd.AddCommand(
        NewWithdrawRewardsCmd(),
        NewWithdrawAllRewardsCmd(),
        NewSetWithdrawAddrCmd(),
        NewFundCommunityPoolCmd(),
        // ... other commands ...
    )
    
    return distTxCmd
}
```

---

### **Phase 3: Update Hooks for Lazy Withdrawal**

#### Step 3.1: Withdraw on Delegation Removal
- **File**: `x/distribution/keeper/hooks.go`
- **Action**: Update hooks to withdraw before removal

```go
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
    // Simply call the withdrawal function
    // This will use all the improved calculation logic
    _, err := h.k.WithdrawDelegationRewards(ctx, delAddr, valAddr)
    if err != nil {
        h.k.Logger(ctx).Error("Failed to withdraw rewards before delegation removal", "error", err)
    }
    return nil
}

func (h Hooks) BeforeNFTDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
    // Same as above - withdrawal covers both native and NFT
    _, err := h.k.WithdrawDelegationRewards(ctx, delAddr, valAddr)
    if err != nil {
        h.k.Logger(ctx).Error("Failed to withdraw NFT delegation rewards before removal", "error", err)
    }
    return nil
}
```

#### Step 3.2: Keep Period Increment Hooks
- **File**: `x/distribution/keeper/hooks.go`
- **Functions**: `BeforeDelegationCreated`, `BeforeNFTDelegationCreated`, etc.
- **Action**: Keep as-is (they're correct for pro-rating)

---

### **Phase 4: Testing**

#### Test Case 1: Manual Withdrawal
```bash
# Delegate tokens
denergyd tx staking delegate <validator> 1000token --from delegator

# Wait for epoch end (rewards allocated)

# Query rewards
denergyd q distribution rewards <delegator> <validator>

# Withdraw rewards
denergyd tx distribution withdraw-rewards <validator> --from delegator

# Verify balance increased
denergyd q bank balances <delegator>
```

#### Test Case 2: Withdraw All
```bash
# Delegate to multiple validators
denergyd tx staking delegate <validator1> 1000token --from delegator
denergyd tx staking delegate <validator2> 1000token --from delegator

# Wait for epoch end

# Withdraw from all validators at once
denergyd tx distribution withdraw-all-rewards --from delegator
```

#### Test Case 3: Withdrawal on Unbonding
```bash
# Delegate and wait for rewards
denergyd tx staking delegate <validator> 1000token --from delegator

# Wait for epoch end

# Unbond (should automatically withdraw)
denergyd tx staking unbond <validator> 1000token --from delegator

# Check that rewards were withdrawn
denergyd q bank balances <delegator>
```

#### Test Case 4: Query Before Withdrawal
```bash
# Query should increment period temporarily and show accurate rewards
denergyd q distribution rewards <delegator> <validator>

# Withdrawal should work correctly
denergyd tx distribution withdraw-rewards <validator> --from delegator
```

---

## 📊 Gas Comparison

### Automatic Distribution (Every Epoch):
```
1000 delegators × 100K gas per distribution = 100M gas per epoch
Every epoch pays this cost regardless of activity
```

### Lazy Withdrawal:
```
Each delegator pays ~100K gas only when they withdraw
Gas distributed over time
Only active users pay gas
```

**Winner**: Lazy withdrawal is much more efficient! ✅

---

## 🔄 Migration Strategy

### Option 1: Clean Break
- Deploy new version with lazy withdrawal
- Document that users must manually claim after upgrade

### Option 2: One Final Auto-Distribution
- Before disabling auto-distribution, run one final distribution
- Then switch to lazy withdrawal
- Users start with zero accumulated rewards

### Option 3: Gradual (Recommended)
- Keep auto-distribution code but add parameter to disable it
- Default: disabled (lazy withdrawal)
- Can be enabled via governance if needed

---

## ✅ Success Criteria

1. **Manual Withdrawal Works**: Delegators can claim rewards via transaction
2. **Accurate Calculation**: Uses improved calculation logic with all bug fixes
3. **Gas Efficient**: No epoch-end mass distribution
4. **Auto-Withdraw on Unbond**: Rewards automatically claimed before delegation removal
5. **Query Accuracy**: Query endpoints show correct pending rewards
6. **Logging Maintained**: Separate NFT/Native breakdown in logs

---

## 📅 Implementation Timeline

| Phase | Tasks | Time |
|-------|-------|------|
| 1 | Remove auto-distribution | 1 day |
| 2 | Restore withdrawal endpoints | 2 days |
| 3 | Update hooks | 1 day |
| 4 | Testing & validation | 2 days |

**Total**: ~1 week

---

## 🎯 Summary

**Removing**: Automatic distribution at epoch end (expensive)
**Keeping**: All calculation improvements, bug fixes, logging
**Adding Back**: Manual withdrawal transaction endpoints
**Result**: Efficient lazy withdrawal with improved accuracy

This gives you the best of both worlds:
- ✅ Efficient gas usage (users pay when they claim)
- ✅ Accurate reward calculation (all bug fixes preserved)
- ✅ Detailed logging (NFT/Native breakdown)
- ✅ Pro-rating support (when implemented)
- ✅ User control (claim when they want)

