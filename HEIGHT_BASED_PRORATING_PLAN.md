# Height-Based Pro-Rating Implementation Plan

## 📋 Overview

Implement true pro-rated rewards based on block heights, ensuring delegators who join mid-epoch receive proportional rewards based on the time they were actively delegated.

---

## 🎯 Goals

1. **Fair Distribution**: Delegators receive rewards proportional to blocks they were delegated
2. **No Gaming**: Can't join right before epoch end and receive full epoch rewards
3. **Accurate Calculation**: Pro-rating based on actual block heights, not just periods
4. **Maintain Efficiency**: Keep O(1) reward calculation (don't iterate through blocks)

---

## 🤔 Do We Still Need Periods?

**YES - Periods are still essential!**

### Why Periods Are Required:

1. **Cumulative Ratio Storage**: Periods store cumulative reward ratios, which is the core of F1 algorithm
2. **O(1) Calculation**: Without periods, we'd need to iterate through all epochs/blocks (O(N))
3. **Multiple Epochs**: A delegator can span multiple epochs, periods track cumulative rewards across them

### What Changes:

- **Before**: Periods alone determined pro-rating (doesn't work for epoch-based allocation)
- **After**: Periods provide cumulative ratios + Block heights provide pro-rating factor

**Example:**
```
Delegator joins at block 1500
Epoch 1 ends at block 2000 (500 blocks active)
Epoch 2 ends at block 3000 (1000 blocks active)

Without height pro-rating:
- Rewards = cumulative_ratio[epoch2] - cumulative_ratio[epoch0]
- Gets full rewards for both epochs ❌

With height pro-rating:
- Epoch 1: (500/1000) * rewards = 50%
- Epoch 2: (1000/1000) * rewards = 100%
- Fair distribution! ✅
```

---

## 📐 Architecture

### Current Data Structures:

```protobuf
// DelegatorStartingInfo (already has Height!)
message DelegatorStartingInfo {
  uint64 previous_period = 1;
  string stake = 2;
  string nft_stake = 3;
  uint64 height = 4;  // ← Block when delegation was created
}

// ValidatorHistoricalRewards (needs Height!)
message ValidatorHistoricalRewards {
  repeated cosmos.base.v1beta1.DecCoin cumulative_reward_ratio = 1;
  repeated cosmos.base.v1beta1.DecCoin nft_cumulative_reward_ratio = 2;
  uint32 reference_count = 3;
  // MISSING: Block height when this period was created
}
```

### Required Changes:

```protobuf
// Add height tracking to historical rewards
message ValidatorHistoricalRewards {
  repeated cosmos.base.v1beta1.DecCoin cumulative_reward_ratio = 1;
  repeated cosmos.base.v1beta1.DecCoin nft_cumulative_reward_ratio = 2;
  uint32 reference_count = 3;
  uint64 height = 4;  // ← NEW: Block height when period was created
}
```

---

## 🔧 Implementation Steps

### **Phase 1: Add Height Tracking to Periods**

#### Step 1.1: Update Proto Definitions
- **File**: `proto/cosmos/distribution/v1beta1/distribution.proto`
- **Action**: Add `height` field to `ValidatorHistoricalRewards`
- **Migration**: Existing records get height = 0 (treated as beginning of chain)

#### Step 1.2: Regenerate Proto Files
- Run `make proto-gen`
- Update generated files in `x/distribution/types/`

#### Step 1.3: Update Period Creation Logic
- **File**: `x/distribution/keeper/validator.go`
- **Function**: `IncrementValidatorPeriod`
- **Action**: Store current block height when creating historical rewards
```go
newHistorical := NewValidatorHistoricalRewardsWithHeight(
    cumulativeRatio, 
    nftCumulativeRatio, 
    1,  // reference count
    ctx.BlockHeight(),  // ← NEW
)
```

#### Step 1.4: Update Validator Initialization
- **File**: `x/distribution/keeper/validator.go`
- **Function**: `initializeValidator`
- **Action**: Set initial period (0) height to current block height

---

### **Phase 2: Implement Pro-Rating Logic**

#### Step 2.1: Create Pro-Rating Calculator
- **File**: `x/distribution/keeper/prorating.go` (NEW)
- **Purpose**: Calculate pro-rating factor for a delegation within a period

```go
// calculateProRatingFactor calculates what fraction of period rewards 
// the delegator should receive based on when they joined
func (k Keeper) calculateProRatingFactor(
    ctx context.Context,
    delegationHeight uint64,  // When delegator joined
    periodStartHeight uint64,  // When period started
    periodEndHeight uint64,    // When period ended
) math.LegacyDec {
    // Delegator was active from max(delegation, periodStart) to periodEnd
    activeStart := max(delegationHeight, periodStartHeight)
    activeBlocks := periodEndHeight - activeStart
    totalBlocks := periodEndHeight - periodStartHeight
    
    if totalBlocks == 0 {
        return math.LegacyOneDec()  // Full rewards if same block
    }
    
    return math.LegacyNewDec(activeBlocks).Quo(math.LegacyNewDec(totalBlocks))
}
```

#### Step 2.2: Update Single Period Reward Calculation
- **File**: `x/distribution/keeper/delegation.go`
- **New Function**: `calculateDelegationRewardsBetweenWithProRating`

```go
// Calculate rewards for a single period with pro-rating
func (k Keeper) calculateDelegationRewardsForPeriod(
    ctx context.Context,
    val stakingtypes.ValidatorI,
    startingPeriod uint64,
    endingPeriod uint64,
    stake math.LegacyDec,
    delegationHeight uint64,
) (sdk.DecCoins, error) {
    // Get historical rewards (contains cumulative ratios AND heights)
    starting, _ := k.GetValidatorHistoricalRewards(ctx, valAddr, startingPeriod)
    ending, _ := k.GetValidatorHistoricalRewards(ctx, valAddr, endingPeriod)
    
    // Calculate base rewards using F1 algorithm
    difference := ending.CumulativeRewardRatio.Sub(starting.CumulativeRewardRatio)
    baseRewards := difference.MulDecTruncate(stake)
    
    // Calculate pro-rating factor
    proRateFactor := k.calculateProRatingFactor(
        ctx,
        delegationHeight,
        starting.Height,
        ending.Height,
    )
    
    // Apply pro-rating
    proRatedRewards := baseRewards.MulDecTruncate(proRateFactor)
    
    return proRatedRewards, nil
}
```

#### Step 2.3: Update Multi-Period Reward Calculation
- **Challenge**: A delegator might span multiple periods/epochs
- **Solution**: Calculate pro-rating for each period separately, then sum

```go
func (k Keeper) calculateDelegationRewardsBetween(
    ctx context.Context,
    val stakingtypes.ValidatorI,
    startingPeriod uint64,
    endingPeriod uint64,
    stake math.LegacyDec,
    delegationHeight uint64,
) (sdk.DecCoins, error) {
    totalRewards := sdk.NewDecCoins()
    
    // Iterate through each period from start to end
    for period := startingPeriod; period < endingPeriod; period++ {
        periodRewards, err := k.calculateDelegationRewardsForPeriod(
            ctx, val, period, period+1, stake, delegationHeight,
        )
        if err != nil {
            return nil, err
        }
        totalRewards = totalRewards.Add(periodRewards...)
    }
    
    return totalRewards, nil
}
```

#### Step 2.4: Update NFT Reward Calculation
- **File**: `x/distribution/keeper/delegation.go`
- **Function**: `calculateNFTDelegationRewardsBetween`
- **Action**: Add same pro-rating logic for NFT rewards

---

### **Phase 3: Update Distribution Functions**

#### Step 3.1: Update Automatic Distribution
- **File**: `x/distribution/keeper/auto_distribution.go`
- **Function**: `distributeRewardsToSingleDelegator`
- **Action**: Pass `delegationHeight` from `DelegatorStartingInfo.Height`

```go
func (k Keeper) distributeRewardsToSingleDelegator(...) {
    startingInfo, _ := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
    
    // Pass delegation height for pro-rating
    nativeRewards, _ := k.calculateDelegationRewardsBetween(
        ctx, val, 
        startingPeriod, endingPeriod, 
        nativeStake,
        startingInfo.Height,  // ← Pro-rating based on this
    )
    
    nftRewards, _ := k.calculateNFTDelegationRewardsBetween(
        ctx, val,
        startingPeriod, endingPeriod,
        nftStake,
        startingInfo.Height,  // ← Pro-rating based on this
    )
    
    // ... rest of distribution
}
```

#### Step 3.2: Update Query Endpoints
- **File**: `x/distribution/keeper/grpc_query.go`
- **Function**: `DelegationRewards`, `DelegationTotalRewards`
- **Action**: Include pro-rating in query responses

---

### **Phase 4: Handle Edge Cases**

#### Step 4.1: Same-Block Period Changes
- **Scenario**: Multiple delegations in same block
- **Solution**: `activeBlocks = 0`, `totalBlocks = 0` → return 100% (full rewards)

#### Step 4.2: Delegation Before Period Start
- **Scenario**: `delegationHeight < periodStartHeight`
- **Solution**: Use `max(delegationHeight, periodStartHeight)` as active start

#### Step 4.3: Migration (Existing Delegators)
- **Scenario**: Existing delegators with no height in historical rewards
- **Solution**: 
  - `height = 0` means "from genesis" → gets full rewards
  - Document that existing delegators keep full rewards until next delegation change

#### Step 4.4: Period 0 Edge Case
- **Scenario**: Initial period (0) might not have meaningful height
- **Solution**: Set period 0 height to genesis block height during `initializeValidator`

---

### **Phase 5: Enhanced Logging**

#### Step 5.1: Add Pro-Rating Logs
```go
k.Logger(ctx).Info("📊 Pro-rating calculation",
    "delegator", delAddr.String(),
    "delegation_height", delegationHeight,
    "period_start_height", periodStartHeight,
    "period_end_height", periodEndHeight,
    "blocks_active", activeBlocks,
    "total_blocks", totalBlocks,
    "pro_rate_factor", proRateFactor.String(),
    "base_rewards", baseRewards.String(),
    "pro_rated_rewards", proRatedRewards.String(),
)
```

#### Step 5.2: Update Distribution Logs
```go
k.Logger(ctx).Info("💰 Calculated rewards",
    "delegator", delAddr.String(),
    "validator", val.GetOperator(),
    "native_rewards", nativeRewards.String(),
    "nft_rewards", nftRewards.String(),
    "total_rewards", totalRewards.String(),
    "delegation_height", startingInfo.Height,
    "was_prorated", startingInfo.Height > periodStartHeight,
)
```

---

## 🧪 Testing Strategy

### Test Case 1: Full Epoch Delegation
```
Delegator joins at block 1000 (epoch start)
Epoch ends at block 2000
Expected: 100% of rewards (1000/1000 blocks)
```

### Test Case 2: Mid-Epoch Delegation
```
Epoch starts at block 1000
Delegator joins at block 1500 (mid-epoch)
Epoch ends at block 2000
Expected: 50% of rewards (500/1000 blocks)
```

### Test Case 3: Late-Epoch Delegation
```
Epoch starts at block 1000
Delegator joins at block 1900 (near end)
Epoch ends at block 2000
Expected: 10% of rewards (100/1000 blocks)
```

### Test Case 4: Multi-Epoch Delegation
```
Epoch 1: Block 1000-2000 (joins at 1500 = 50%)
Epoch 2: Block 2000-3000 (active full epoch = 100%)
Expected: 50% of Epoch 1 + 100% of Epoch 2 rewards
```

### Test Case 5: Same Block Delegation
```
Multiple delegators join at same block
Expected: All get same pro-rated amount
```

---

## 📊 Example Calculation Walkthrough

### Scenario:
- Epoch starts: Block 1000
- Validator gets 1000 tokens reward at end of epoch
- Validator has 1000 total tokens staked

**Delegator A:**
- Joins at block 1000 (epoch start)
- Stakes 500 tokens (50% of total)

**Delegator B:**
- Joins at block 1500 (mid-epoch)
- Stakes 500 tokens (50% of total)

**Epoch ends: Block 2000**

### Current Behavior (NO Pro-Rating):
```
Period 0 → Period 1 (rewards allocated)

Delegator A:
- Base rewards: 500 * (cumRatio[1] - cumRatio[0]) = 500 tokens
- Gets 50% of total ✅

Delegator B:
- Base rewards: 500 * (cumRatio[1] - cumRatio[0]) = 500 tokens
- Gets 50% of total ❌ (only delegated for 50% of epoch!)
```

### With Pro-Rating:
```
Delegator A:
- Base rewards: 500 tokens
- Pro-rate factor: (2000-1000)/(2000-1000) = 1.0
- Final rewards: 500 * 1.0 = 500 tokens ✅

Delegator B:
- Base rewards: 500 tokens
- Pro-rate factor: (2000-1500)/(2000-1000) = 0.5
- Final rewards: 500 * 0.5 = 250 tokens ✅

Total distributed: 750 tokens
Remaining: 250 tokens for validator commission
```

---

## 🚨 Breaking Changes

### Proto Changes:
- `ValidatorHistoricalRewards` gains `height` field
- This is a **state-breaking change**
- Requires coordination with validators

### Migration Strategy:
1. **Soft Fork**: Add field with default `height = 0`
2. **Interpretation**: `height = 0` means "before pro-rating" → full rewards
3. **Gradual Adoption**: New periods have height, old periods treated as full-epoch
4. **No Data Loss**: Existing rewards continue to work

---

## ✅ Success Criteria

1. **Accurate Pro-Rating**: Rewards proportional to blocks delegated
2. **No Gaming**: Late joiners don't get full rewards
3. **Performance**: Maintains O(1) calculation (no block iteration)
4. **Multi-Epoch Support**: Works across multiple epochs
5. **Edge Case Handling**: Same-block, migration, period 0 all work correctly
6. **Clear Logging**: Easy to audit and debug reward calculations

---

## 📅 Implementation Timeline

### Week 1: Foundation
- Phase 1: Proto changes and height tracking
- Proto regeneration and testing

### Week 2: Core Logic
- Phase 2: Pro-rating calculation implementation
- Unit tests for pro-rating logic

### Week 3: Integration
- Phase 3: Update distribution and queries
- Integration tests for full flow

### Week 4: Polish & Testing
- Phase 4: Edge cases and migration
- Phase 5: Enhanced logging
- End-to-end testing

---

## 🎯 Next Steps

1. **Review this plan** - Confirm approach is acceptable
2. **Phase 1 Implementation** - Start with proto changes
3. **Test on devnet** - Verify height tracking works
4. **Phase 2 Implementation** - Add pro-rating logic
5. **Comprehensive testing** - All test cases pass
6. **Deploy to testnet** - Real-world validation
7. **Production deployment** - Coordinated upgrade

---

## ❓ Open Questions

1. **Migration Timeline**: When to activate pro-rating for existing delegators?
2. **Validator Commission**: Should commission also be pro-rated based on validator uptime?
3. **Rounding**: How to handle dust amounts from pro-rating fractions?
4. **Audit**: Should we audit pro-rating calculations with third party?

---

## 📝 Notes

- Pro-rating is calculated **per period**, not per epoch (supports multi-epoch delegations)
- Block heights stored in periods enable efficient O(1) pro-rating calculation
- Existing F1 algorithm remains intact, pro-rating is an additional factor
- Periods are still essential for cumulative ratio storage and efficient calculation

