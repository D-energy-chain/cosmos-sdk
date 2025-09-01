# DEnergy Rewards Distribution System

## Overview

The DEnergy blockchain implements a sophisticated dual-staking rewards distribution system that supports both NFT staking and native token staking. The system operates on an **epoch-based distribution model** with **zero community tax**, ensuring maximum rewards flow directly to validators and delegators.

## Key Features

- **Dual Staking Support**: Both NFT and native token staking
- **Epoch-Based Distribution**: Rewards accumulated and distributed at epoch boundaries
- **Zero Community Tax**: 100% of collected fees go to validators and delegators
- **Configurable Ratios**: Flexible allocation between NFT and native staking rewards
- **F1 Distribution**: Uses proven Cosmos SDK F1 fee distribution mechanism

## Distribution Architecture

### Core Components

1. **Fee Collection**: Transaction fees accumulated in the fee collector module during each epoch
2. **Epoch-Based Processing**: Rewards distributed only at epoch end via `AfterEpochEnd` hook
3. **Dual Allocation**: Rewards split between NFT stakers and native token stakers
4. **Validator Commission**: Applied uniformly across both staking types

### Distribution Flow

```
Collected Fees (100%)
    │
    └── Validators (100% - 0% community tax)
        │
        ├── Validator Commission (configurable %)
        │   ├── NFT Staking Commission
        │   └── Native Staking Commission
        │
        └── Delegator Rewards
            ├── NFT Staking Rewards (75% default)
            └── Native Staking Rewards (25% default)
```

## Default Configuration

### Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| `community_tax` | 0.00 (0%) | No community tax - all fees to validators |
| `nft_staking_ratio` | 0.75 (75%) | Portion of staking rewards for NFT stakers |
| `native_staking_ratio` | 0.25 (25%) | Portion of staking rewards for native stakers |
| `withdraw_addr_enabled` | true | Allow custom withdrawal addresses |

### Validation Rules

- `nft_staking_ratio + native_staking_ratio = 1.00` (100% of staking rewards)
- All ratios must be between 0.00 and 1.00
- No community tax deduction (0%)

## How It Works

### 1. Fee Collection Phase

During each epoch, transaction fees are collected in the fee collector module account:
- Gas fees from transactions
- Any other protocol fees
- Fees remain in collector until epoch end

### 2. Epoch-Based Distribution Trigger

At epoch end, the `AfterEpochEnd` hook in `hooks.go:191-224` triggers:

```go
func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
    // Calculate total voting power
    var previousTotalPower int64
    for _, voteInfo := range ctx.VoteInfos() {
        previousTotalPower += voteInfo.Validator.Power
    }
    
    // Distribute all accumulated rewards
    if err := h.k.AllocateTokens(ctx, previousTotalPower, ctx.VoteInfos()); err != nil {
        // Handle error
    }
}
```

### 3. Token Allocation Process

The `AllocateTokens` function (`allocation.go:18-133`):

1. **Fetches accumulated fees** from fee collector
2. **Transfers fees** to distribution module 
3. **Distributes proportionally** to validators based on voting power
4. **Handles missed votes** by redirecting rewards to community pool

### 4. Validator-Level Distribution

For each validator, `AllocateTokensToValidator` (`allocation.go:137-255`):

1. **Retrieves share information**:
   - NFT delegator shares: `val.GetNFTDelegatorShares()`
   - Native delegator shares: `val.GetDelegatorShares()`

2. **Applies allocation logic**:

#### Case A: Both NFT and Native Staking
```go
// Split based on configured ratios
nftRewards = tokens.MulDecTruncate(nftStakingRatio)     // 75%
nativeRewards = tokens.MulDecTruncate(nativeStakingRatio) // 25%

// Apply commission to each type
nftCommission = nftRewards.MulDec(val.GetCommission())
nativeCommission = nativeRewards.MulDec(val.GetCommission())
```

#### Case B: Only NFT Staking
```go
// All rewards go to NFT stakers
nftRewards = tokens
nftCommission = tokens.MulDec(val.GetCommission())
```

#### Case C: Only Native Staking  
```go
// All rewards go to native stakers
nativeRewards = tokens
nativeCommission = tokens.MulDec(val.GetCommission())
```

3. **Updates reward tracking**:
   - Validator accumulated commission
   - Current validator rewards
   - Outstanding rewards

## Detailed Examples

### Example 1: Standard Dual Staking Distribution

**Scenario**:
- Total epoch fees: 10,000 tokens
- Validator with 10% voting power receives: 1,000 tokens
- Validator commission: 8%
- Validator has both NFT and native delegations

**Distribution**:
```
1. Validator receives: 1,000 tokens (10% of total fees)

2. NFT Staking Allocation (75%):
   - NFT rewards: 1,000 × 0.75 = 750 tokens
   - Validator commission: 750 × 0.08 = 60 tokens
   - NFT delegators: 750 - 60 = 690 tokens

3. Native Staking Allocation (25%):
   - Native rewards: 1,000 × 0.25 = 250 tokens
   - Validator commission: 250 × 0.08 = 20 tokens
   - Native delegators: 250 - 20 = 230 tokens

Final Distribution:
- Validator total commission: 80 tokens (8%)
- NFT delegators: 690 tokens (69%)
- Native delegators: 230 tokens (23%)
```

### Example 2: NFT-Only Validator

**Scenario**:
- Validator rewards: 1,000 tokens
- Validator has only NFT delegations (no native staking)
- Commission: 5%

**Distribution**:
```
1. All rewards go to NFT stakers: 1,000 tokens
2. Validator commission: 1,000 × 0.05 = 50 tokens
3. NFT delegators: 1,000 - 50 = 950 tokens

Final Distribution:
- Validator commission: 50 tokens (5%)
- NFT delegators: 950 tokens (95%)
```

### Example 3: Native-Only Validator

**Scenario**:
- Validator rewards: 1,000 tokens
- Validator has only native token delegations
- Commission: 12%

**Distribution**:
```
1. All rewards go to native stakers: 1,000 tokens
2. Validator commission: 1,000 × 0.12 = 120 tokens
3. Native delegators: 1,000 - 120 = 880 tokens

Final Distribution:
- Validator commission: 120 tokens (12%)
- Native delegators: 880 tokens (88%)
```

## Edge Cases and Special Scenarios

### Edge Case 1: Validator with No Delegations

**Scenario**: Validator has no delegator shares (neither NFT nor native)

**Behavior**:
- Function exits early with no reward allocation
- Rewards remain in fee pool (logged as warning)
- Code: `allocation.go:152-157`

### Edge Case 2: Missed Block Votes

**Scenario**: Validator fails to sign the block (BlockIDFlagCommit != commit)

**Behavior**:
- Validator skipped in distribution process
- Their share of rewards effectively goes to community pool
- Code: `allocation.go:82-87`

### Edge Case 3: Zero Total Voting Power

**Scenario**: No validators participated in consensus

**Behavior**:
- All collected fees sent directly to community pool
- No validator rewards distributed
- Code: `allocation.go:59-63`

### Edge Case 4: Precision Truncation

**Scenario**: Reward calculations result in decimal amounts

**Behavior**:
- Rewards truncated to integer amounts before distribution
- Remainder dust sent to community pool
- Maintains invariant that no tokens are lost

### Edge Case 5: Commission Rate Changes

**Scenario**: Validator changes commission during epoch

**Behavior**:
- New commission rate applies to current epoch rewards
- Historical rewards maintain original commission rates
- Handled through F1 distribution mechanism

## Epoch Integration

### Epoch Keeper Interface

The system integrates with the parent codebase's epoch module via:

```go
type EpochKeeper interface {
    AllEpochInfos(ctx sdk.Context) []EpochInfo
    IsEpochEnd(ctx sdk.Context, identifier string) bool
}
```

### Epoch End Processing

1. **Proposer Recording**: Current block proposer recorded for next distribution
2. **Power Calculation**: Total voting power calculated from vote info
3. **Reward Distribution**: All accumulated fees distributed proportionally
4. **State Updates**: Validator and delegator reward states updated

## Benefits of This System

### 1. Computational Efficiency
- Rewards processed once per epoch vs. every block
- Reduces validator computational overhead
- Lower gas costs for reward-related operations

### 2. Predictable Distribution
- Users know exactly when rewards are distributed
- Easier planning for delegation strategies
- Clear epoch-based accounting

### 3. Dual Asset Support
- Supports both NFT and native token economies
- Flexible ratio configuration
- Independent commission structures

### 4. Zero Tax Advantage
- 100% of fees return to network participants
- No community pool deduction
- Maximum incentive alignment

## Technical Implementation Details

### Key Files

- `keeper/allocation.go`: Core distribution logic
- `keeper/hooks.go`: Epoch integration and lifecycle management
- `types/params.go`: Parameter definitions and validation
- Documentation: `NFT_STAKING_REWARDS.md`, `DUAL_STAKING_REWARDS.md`

### State Management

- **FeePool**: Tracks community pool balances
- **ValidatorAccumulatedCommission**: Per-validator commission tracking
- **ValidatorCurrentRewards**: Current period rewards
- **ValidatorOutstandingRewards**: Total outstanding rewards
- **DelegatorStartingInfo**: F1 distribution tracking

### Integration Points

1. **Staking Module**: Provides validator and delegation information
2. **Bank Module**: Handles token transfers
3. **Epoch Module**: Triggers distribution at epoch boundaries
4. **Auth Module**: Manages module accounts

## Security Considerations

### Invariants Maintained

1. **Token Conservation**: No tokens created or destroyed in distribution
2. **Commission Bounds**: Commission rates respected and enforced
3. **Share Consistency**: Reward shares match actual delegation shares
4. **Epoch Atomicity**: Distribution is atomic per epoch

### Attack Vectors Mitigated

1. **Commission Manipulation**: Commission changes apply to future rewards only
2. **Share Dilution**: F1 distribution prevents historical share manipulation
3. **Rounding Exploitation**: Truncation dust sent to community pool
4. **Validator Centralization**: Proportional distribution maintains decentralization

## Monitoring and Observability

### Key Metrics

- Total fees collected per epoch
- Distribution completion time
- Validator participation rates
- Commission distribution ratios
- NFT vs native staking ratios

### Logging

Comprehensive logging throughout the distribution process:
- Fee collection amounts (`allocation.go:20-33`)
- Validator reward calculations (`allocation.go:100-104`)
- Distribution completion (`hooks.go:223`)

## Future Considerations

### Parameter Adjustability

- Staking ratios can be modified via governance
- Commission structures could be enhanced
- Additional asset types could be supported

### Performance Optimization

- Batch processing for large validator sets
- Parallel reward calculations
- Optimized state access patterns

This distribution system provides a robust, efficient, and fair mechanism for rewarding network participants while maintaining the flexibility to adapt to evolving tokenomics requirements.