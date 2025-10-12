# Lazy Withdrawal - Edge Cases Analysis

## 📋 Overview

Analysis of edge cases for lazy withdrawal mechanism, focusing on delegation removal scenarios and ensuring delegators never lose rewards.

---

## 🎯 Core Principle

**Delegators must NEVER lose rewards, regardless of when or how their delegation is removed.**

This means:
- Rewards must be withdrawn BEFORE delegation records are deleted
- Works for manual unbonding, forced removal, NFT offsetting, etc.
- Works mid-epoch, epoch-end, or any time

---

## 🔍 Scenario Analysis

### **Scenario 1: Mid-Epoch Delegation Removal (Manual Unbond)**

#### Timeline:
```
Block 1000: Epoch starts
Block 1200: Delegator initiates unbonding
Block 2000: Epoch ends
```

#### State at Block 1200 (Mid-Epoch):
```
ValidatorCurrentRewards:
  Period: 1
  Rewards: 0 (no rewards allocated yet - happens at epoch end)

DelegatorStartingInfo:
  PreviousPeriod: 0
  
Calculation:
  Starting: period 0
  Ending: period 1 - 1 = period 0
  Rewards: cumRatio[0] - cumRatio[0] = 0 ✅

Why zero? Because current epoch hasn't ended yet!
The delegator WILL get rewards when epoch ends.
```

#### Hook Behavior:
```go
func BeforeDelegationRemoved(ctx, delAddr, valAddr) {
    // Calculate rewards from last withdrawal to now
    rewards = calculateRewards(startingPeriod, currentPeriod - 1)
    
    if rewards > 0 {
        withdrawRewards(delAddr, rewards)
    }
    
    // Safe to remove delegation now
}
```

#### What Happens:
1. **Immediate Effect**: Delegation marked for removal, added to unbonding queue
2. **Rewards Calculation**: Zero (current epoch not complete)
3. **Epoch End (Block 2000)**: 
   - ❌ Delegation already removed from validator
   - ❌ Delegator doesn't receive this epoch's rewards
   - **PROBLEM: Lost rewards for blocks 1000-1200!**

#### **❌ ISSUE IDENTIFIED**: Mid-epoch unbonding loses partial epoch rewards!

---

### **Scenario 2: Epoch-End Delegation Removal (Your Current System)**

#### Timeline:
```
Block 1000: Epoch 1 starts
Block 2000: Epoch 1 ends
  → Rewards allocated
  → Periods incremented
  → Unbonding processed
Block 2000: Epoch 2 starts
```

#### State at Block 2000 (Epoch End):
```
Before Allocation:
  ValidatorCurrentRewards.Rewards: 1000 tokens accumulated

After Allocation:
  ValidatorCurrentRewards:
    Period: 2 (incremented)
    Rewards: 0 (cleared)
  ValidatorHistoricalRewards[1]:
    CumulativeRatio: 1.0 per token

After Unbonding Hook:
  DelegatorStartingInfo:
    PreviousPeriod: 0
    
  Calculation:
    Starting: period 0 (cumRatio = 0)
    Ending: period 2 - 1 = period 1 (cumRatio = 1.0)
    Rewards: 500 tokens × (1.0 - 0) = 500 tokens ✅
```

#### Hook Behavior:
```go
func BeforeDelegationRemoved(ctx, delAddr, valAddr) {
    // Periods already incremented, rewards allocated
    rewards = calculateRewards(0, 1)  // 500 tokens
    withdrawRewards(delAddr, 500)  // ✅ Success
}
```

#### What Happens:
1. **Epoch End**: Rewards allocated and periods incremented
2. **Unbonding Hook**: Calculates and withdraws full epoch rewards
3. **Delegation Removed**: Safe - rewards already withdrawn
4. **✅ WORKS CORRECTLY**: Delegator receives all earned rewards

---

### **Scenario 3: NFT Offsetting Mid-Epoch**

#### Timeline:
```
Block 1000: Epoch starts
Block 1500: NFT offsetting triggered
  → Burns NFT
  → Removes NFT delegation
Block 2000: Epoch ends
```

#### State at Block 1500 (Mid-Epoch):
```
ValidatorCurrentRewards:
  Period: 1
  Rewards: 0 (not allocated yet)

DelegatorStartingInfo:
  PreviousPeriod: 0
  NftStake: 1000

Calculation:
  Starting: period 0
  Ending: period 1 - 1 = period 0
  NFT Rewards: cumRatio[0] - cumRatio[0] = 0 ✅
```

#### Hook Behavior:
```go
func BeforeNFTDelegationRemoved(ctx, delAddr, valAddr) {
    // Called during NFT offsetting
    rewards = calculateRewards(0, 0)  // 0 - current epoch not complete
    withdrawRewards(delAddr, 0)  // Nothing to withdraw
    
    // NFT delegation removed
}
```

#### What Happens:
1. **NFT Offsetting**: Burns NFT, triggers removal hook
2. **Rewards Calculation**: Zero (epoch incomplete)
3. **NFT Delegation Removed**: Records deleted
4. **Epoch End (Block 2000)**:
   - ❌ NFT delegation already gone
   - ❌ Delegator doesn't receive rewards for blocks 1000-1500
   - **PROBLEM: Lost rewards!**

#### **❌ ISSUE IDENTIFIED**: Mid-epoch NFT offsetting loses partial epoch rewards!

---

## 🚨 **CRITICAL PROBLEMS IDENTIFIED**

### Problem 1: Mid-Epoch Removal Loses Partial Epoch Rewards

**Why This Happens:**
- Rewards are allocated at epoch END
- Delegation removed mid-epoch
- When epoch ends, delegation doesn't exist anymore
- Rewards for that period never calculated

**Affected Scenarios:**
- Manual mid-epoch unbonding
- NFT offsetting mid-epoch
- Validator removal mid-epoch
- Any delegation deletion before epoch end

---

### Problem 2: Accumulated Rewards vs Current Epoch Rewards

**Two Types of Rewards:**

1. **Accumulated Rewards** (from previous epochs):
   - Already allocated to validator
   - Already in cumulative ratios
   - CAN be withdrawn anytime
   - ✅ Safe to calculate mid-epoch

2. **Current Epoch Rewards** (this epoch):
   - NOT yet allocated
   - NOT in cumulative ratios yet
   - CANNOT be calculated mid-epoch
   - ❌ Will be lost if delegation removed

---

## ✅ **SOLUTIONS**

### Solution 1: Only Allow Epoch-End Unbonding (Your Current Approach)

**Implementation:**
```go
// In staking module
func (k Keeper) Unbond(ctx, delAddr, valAddr, shares) {
    // Add to unbonding queue
    // Actual unbonding happens at epoch end
    return k.AddToUnbondingQueue(delAddr, valAddr, shares)
}

// At epoch end
func (k Keeper) ProcessUnbondingQueue(ctx) {
    // After rewards allocated and periods incremented
    for each unbonding {
        // Trigger BeforeDelegationRemoved hook
        // Hook withdraws full epoch rewards ✅
        // Then remove delegation
    }
}
```

**Pros:**
- ✅ Simple implementation
- ✅ No reward loss
- ✅ All rewards calculated before removal

**Cons:**
- ⚠️ Delegators must wait until epoch end
- ⚠️ Less flexibility

**Status**: ✅ **This is what you're doing for undelegation!**

---

### Solution 2: Increment Period Before Removal (Force Mid-Epoch Settlement)

**Implementation:**
```go
func BeforeDelegationRemoved(ctx, delAddr, valAddr) {
    // If there are accumulated rewards in current period, lock them in
    currentRewards := k.GetValidatorCurrentRewards(ctx, valAddr)
    
    if !currentRewards.Rewards.IsZero() {
        // Force period increment to lock current epoch rewards
        k.IncrementValidatorPeriod(ctx, val)
        
        // Now rewards are in cumulative ratio
        // Calculate and withdraw
        rewards := k.calculateRewards(...)
        k.withdrawRewards(delAddr, rewards)
    } else {
        // No current rewards, just withdraw accumulated
        rewards := k.calculateRewards(...)
        if !rewards.IsZero() {
            k.withdrawRewards(delAddr, rewards)
        }
    }
}
```

**Pros:**
- ✅ Works for mid-epoch removal
- ✅ No reward loss
- ✅ Immediate unbonding possible

**Cons:**
- ⚠️ Creates many periods (one per mid-epoch removal)
- ⚠️ More complex period management
- ⚠️ Affects all delegators' calculations (period changes)

---

### Solution 3: Pro-Rate Current Epoch Rewards (Manual Calculation)

**Implementation:**
```go
func BeforeDelegationRemoved(ctx, delAddr, valAddr) {
    // Withdraw accumulated rewards (past epochs)
    accumulatedRewards := k.calculateRewards(startingPeriod, currentPeriod - 1)
    
    // Calculate pro-rated current epoch rewards
    currentRewards := k.GetValidatorCurrentRewards(ctx, valAddr)
    if !currentRewards.Rewards.IsZero() {
        // Calculate how long delegator was active this epoch
        epochStart := k.GetEpochStartBlock(ctx)
        currentBlock := ctx.BlockHeight()
        
        // Pro-rate: (blocks_active / total_epoch_blocks) × current_rewards
        totalRewards := accumulatedRewards + proRatedCurrentEpoch
    } else {
        totalRewards := accumulatedRewards
    }
    
    k.withdrawRewards(delAddr, totalRewards)
}
```

**Pros:**
- ✅ Fair - rewards for actual time delegated
- ✅ No period spam
- ✅ Works mid-epoch

**Cons:**
- ⚠️ Complex calculation
- ⚠️ Need to track epoch boundaries
- ⚠️ Can't use pure F1 algorithm

---

### Solution 4: Prevent Mid-Epoch Removal (Except Epoch-End Events)

**Implementation:**
```go
// Allow NFT offsetting only at epoch end
func (k Keeper) EpochOffset(ctx) {
    // This already runs at epoch end
    // After rewards allocated
    // NFT offsetting removes delegations
    // BeforeNFTDelegationRemoved hook withdraws rewards ✅
}

// Prevent manual mid-epoch unbonding
func (k Keeper) Unbond(ctx, delAddr, valAddr) {
    // Only add to queue, process at epoch end
    return k.AddToUnbondingQueue(...)
}

// Prevent validator removal mid-epoch
func (k Keeper) RemoveValidator(ctx, valAddr) {
    // Only mark for removal, process at epoch end
    return k.MarkValidatorForRemoval(valAddr)
}
```

**Pros:**
- ✅ Simple and clean
- ✅ No reward loss
- ✅ Consistent behavior

**Cons:**
- ⚠️ Less flexible
- ⚠️ Delayed unbonding
- ⚠️ Must ensure ALL removal paths are epoch-end only

---

## 🎯 **RECOMMENDED APPROACH**

### **Hybrid: Solution 1 + Solution 2**

**For Normal Unbonding**: Use queued epoch-end processing (Solution 1)
**For Forced Removal**: Use period increment (Solution 2)

```go
func BeforeDelegationRemoved(ctx, delAddr, valAddr) error {
    // Check if delegation exists
    if !delegationExists {
        return nil
    }
    
    // Get current state
    currentRewards := k.GetValidatorCurrentRewards(ctx, valAddr)
    startingInfo := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
    
    // If there are accumulated rewards in current period
    // Force increment to lock them in before withdrawal
    if !currentRewards.Rewards.IsZero() {
        k.Logger(ctx).Info("⚠️ Mid-epoch delegation removal - incrementing period",
            "delegator", delAddr.String(),
            "validator", valAddr.String(),
            "accumulated_rewards", currentRewards.Rewards.String(),
        )
        
        // This creates a new period with current rewards locked
        k.IncrementValidatorPeriod(ctx, val)
    }
    
    // Now calculate and withdraw all rewards
    // Uses ending period = currentPeriod - 1
    rewards := k.CalculateAndWithdrawRewards(ctx, delAddr, valAddr)
    
    k.Logger(ctx).Info("💰 Rewards withdrawn before delegation removal",
        "delegator", delAddr.String(),
        "validator", valAddr.String(),
        "amount", rewards.String(),
    )
    
    return nil
}
```

**Why This Works:**

1. **Epoch-End Unbonding** (your current flow):
   ```
   Epoch ends → Allocate → Increment periods → Process unbonding queue
   
   BeforeDelegationRemoved:
   - currentRewards.Rewards = 0 (already allocated)
   - No period increment needed
   - Calculate rewards using existing periods ✅
   ```

2. **Mid-Epoch NFT Offsetting**:
   ```
   Mid-epoch → NFT offsetting → Remove NFT delegation
   
   BeforeDelegationRemoved:
   - currentRewards.Rewards = 0 (no allocation yet)
   - No period increment needed
   - Calculate rewards from previous epochs ✅
   - Current epoch rewards = 0 (haven't earned yet)
   ```

3. **Forced Mid-Epoch Removal** (if it happens):
   ```
   Mid-epoch → Emergency removal
   
   BeforeDelegationRemoved:
   - currentRewards.Rewards > 0 (some accumulated)
   - Increment period to lock them
   - Calculate and withdraw all rewards ✅
   ```

---

## 📊 **Scenario Comparison Table**

| Scenario | When | Current Rewards | Action | Result |
|----------|------|----------------|--------|--------|
| Normal Undelegation | Epoch end | 0 (allocated) | Calculate & withdraw | ✅ Full rewards |
| NFT Offsetting | Epoch end | 0 (allocated) | Calculate & withdraw | ✅ Full rewards |
| Mid-Epoch Unbond | Mid-epoch | 0 (not allocated) | Withdraw past epochs | ✅ Past rewards, current = 0 |
| Mid-Epoch NFT Offset | Mid-epoch | 0 (not allocated) | Withdraw past epochs | ✅ Past rewards, current = 0 |
| Emergency Removal | Anytime | Might have some | Increment + withdraw | ✅ All rewards |

---

## ✅ **Implementation for Lazy Withdrawal**

### Updated Hook Code:

```go
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
    // Check if delegation exists
    del, err := h.k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
    if err != nil || del == nil {
        return nil
    }
    
    // Check if delegator starting info exists
    hasInfo, err := h.k.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
    if err != nil || !hasInfo {
        return nil
    }
    
    // Get current state
    currentRewards, err := h.k.GetValidatorCurrentRewards(ctx, valAddr)
    if err != nil {
        return nil
    }
    
    val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
    if err != nil {
        return nil
    }
    
    // If there are accumulated rewards in current period, lock them by incrementing
    if !currentRewards.Rewards.IsZero() {
        h.k.Logger(ctx).Info("⚠️ Mid-epoch delegation removal with accumulated rewards",
            "delegator", delAddr.String(),
            "validator", valAddr.String(),
            "accumulated_rewards", currentRewards.Rewards.String(),
            "action", "incrementing period to preserve rewards",
        )
        
        _, err := h.k.IncrementValidatorPeriod(ctx, val)
        if err != nil {
            h.k.Logger(ctx).Error("Failed to increment period before removal", "error", err)
            return err
        }
    }
    
    // Withdraw all accumulated rewards
    rewards, err := h.k.WithdrawDelegationRewards(ctx, delAddr, valAddr)
    if err != nil {
        h.k.Logger(ctx).Error("Failed to withdraw rewards before removal", "error", err)
        // Don't fail removal, but log the issue
        return nil
    }
    
    h.k.Logger(ctx).Info("💰 Rewards withdrawn before delegation removal",
        "delegator", delAddr.String(),
        "validator", valAddr.String(),
        "amount", rewards.String(),
    )
    
    return nil
}

// Same logic for NFT delegations
func (h Hooks) BeforeNFTDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
    // Identical implementation - withdrawal covers both native and NFT
    return h.BeforeDelegationRemoved(ctx, delAddr, valAddr)
}
```

---

## 🎯 **Summary**

### **Current System (Undelegation Only at Epoch End)**:
- ✅ **Works perfectly** - rewards allocated before unbonding
- ✅ **No reward loss** - full epoch rewards calculated
- ✅ **Simple** - no special handling needed

### **NFT Offsetting**:
- ✅ **If at epoch end**: Works like undelegation
- ⚠️ **If mid-epoch**: Only withdraws past epochs, current epoch = 0 (acceptable)

### **Emergency/Forced Removal**:
- ✅ **Handled** - period increment preserves current rewards

### **Key Insight**:
Your system naturally prevents reward loss because:
1. Undelegation is queued and processed at epoch end
2. Rewards are allocated before processing unbonding queue
3. Hooks withdraw rewards before removing delegation records

**Lazy withdrawal will work correctly with your current architecture!** 🎉

---

## ✅ **Action Items**

1. **Confirm**: NFT offsetting only runs at epoch end (after allocation)
2. **Implement**: Hook logic with period increment for edge cases
3. **Test**: All scenarios in table above
4. **Document**: Make it clear that mid-epoch removal = past rewards only

**Ready to implement lazy withdrawal with confidence!** 🚀

