# Reward Distribution Examples

This document provides detailed numerical examples of how rewards are distributed in the dual staking system. Each example includes step-by-step calculations to demonstrate the flow of rewards.

## Default Configuration

Default parameters:

- Community Tax: 20% (0.20)
- NFT Staking Ratio: 75% (0.75) of staking rewards
- Native Staking Ratio: 25% (0.25) of staking rewards
- Separate Pool Ratio: 5% (0.05)

## Example 1: Basic Reward Distribution

Let's say a validator receives 1000 tokens in rewards:

```
Total Rewards: 1000 tokens

1. Community Tax (20%)
   - 1000 * 0.20 = 200 tokens to community pool

2. Separate Pool (5%)
   - 1000 * 0.05 = 50 tokens to separate pool

3. Staking Rewards (75% of remaining tokens)
   - Remaining tokens: 1000 - 200 - 50 = 750 tokens
   - 750 tokens for staking rewards

4. NFT Stakers (75% of staking rewards)
   - 750 * 0.75 = 562.5 tokens to NFT stakers

5. Native Stakers (25% of staking rewards)
   - 750 * 0.25 = 187.5 tokens to native stakers
```

Final Distribution:

- Community Pool: 200 tokens (20%)
- Separate Pool: 50 tokens (5%)
- NFT Stakers: 562.5 tokens (56.25%)
- Native Stakers: 187.5 tokens (18.75%)
- Total: 1000 tokens (100%)

## Example 2: Reward Distribution with Commission

Assume:

- Total Rewards: 1000 tokens
- Validator Commission: 10% (0.10)

```
1. Community Tax (20%)
   - 1000 * 0.20 = 200 tokens

2. Separate Pool (5%)
   - 1000 * 0.05 = 50 tokens

3. Staking Rewards (75% of remaining tokens)
   - Remaining tokens: 1000 - 200 - 50 = 750 tokens
   - 750 tokens for staking rewards

   NFT Stakers (75% of staking rewards = 562.5 tokens):
   - Validator Commission: 562.5 * 0.10 = 56.25 tokens
   - Delegators Share: 562.5 - 56.25 = 506.25 tokens

   Native Stakers (25% of staking rewards = 187.5 tokens):
   - Validator Commission: 187.5 * 0.10 = 18.75 tokens
   - Delegators Share: 187.5 - 18.75 = 168.75 tokens

4. Total Commission: 56.25 + 18.75 = 75 tokens
```

Final Distribution:

- Community Pool: 200 tokens (20%)
- Separate Pool: 50 tokens (5%)
- Validator Commission: 75 tokens (7.5%)
- NFT Delegators: 506.25 tokens (50.625%)
- Native Delegators: 168.75 tokens (16.875%)
- Total: 1000 tokens (100%)

## Example 3: Different Parameter Configuration

Let's use a different configuration:

- Community Tax: 15% (0.15)
- NFT Staking Ratio: 50% (0.50)
- Native Staking Ratio: 50% (0.50)
- Separate Pool Ratio: 10% (0.10)
- Total Rewards: 1000 tokens
- Validator Commission: 5% (0.05)

```
1. Community Tax (15%)
   - 1000 * 0.15 = 150 tokens

2. Separate Pool (10%)
   - 1000 * 0.10 = 100 tokens

3. Staking Rewards (75% of remaining tokens)
   - Remaining tokens: 1000 - 150 - 100 = 750 tokens
   - 750 tokens for staking rewards

   NFT Stakers (50% of staking rewards = 375 tokens):
   - Validator Commission: 375 * 0.05 = 18.75 tokens
   - Delegators Share: 375 - 18.75 = 356.25 tokens

   Native Stakers (50% of staking rewards = 375 tokens):
   - Validator Commission: 375 * 0.05 = 18.75 tokens
   - Delegators Share: 375 - 18.75 = 356.25 tokens

4. Total Commission: 18.75 + 18.75 = 37.5 tokens
```

Final Distribution:

- Community Pool: 150 tokens (15%)
- Separate Pool: 100 tokens (10%)
- Validator Commission: 37.5 tokens (3.75%)
- NFT Delegators: 356.25 tokens (35.625%)
- Native Delegators: 356.25 tokens (35.625%)
- Total: 1000 tokens (100%)

## Example 4: Edge Cases

### Case A: No NFT Stakers

Assume:

- Total Rewards: 1000 tokens
- No NFT shares
- Default parameters

```
1. Community Tax (20%)
   - 1000 * 0.20 = 200 tokens

2. Separate Pool (5%)
   - 1000 * 0.05 = 50 tokens

3. Staking Rewards (750 tokens)
   - All 750 tokens go to native stakers (since no NFT shares exist)
   - Validator Commission (10%): 750 * 0.10 = 75 tokens
   - Native Delegators: 750 - 75 = 675 tokens
```

Final Distribution:

- Community Pool: 200 tokens (20%)
- Separate Pool: 50 tokens (5%)
- Validator Commission: 75 tokens (7.5%)
- Native Delegators: 675 tokens (67.5%)

### Case B: No Native Stakers

Assume:

- Total Rewards: 1000 tokens
- No native shares
- Default parameters

```
1. Community Tax (20%)
   - 1000 * 0.20 = 200 tokens

2. Separate Pool (5%)
   - 1000 * 0.05 = 50 tokens

3. Staking Rewards (750 tokens)
   - All 750 tokens go to NFT stakers (since no native shares exist)
   - Validator Commission (10%): 750 * 0.10 = 75 tokens
   - NFT Delegators: 750 - 75 = 675 tokens
```

Final Distribution:

- Community Pool: 200 tokens (20%)
- Separate Pool: 50 tokens (5%)
- Validator Commission: 75 tokens (7.5%)
- NFT Delegators: 675 tokens (67.5%)

## Example 5: Multiple Validators

Assume:

- Total Rewards: 1000 tokens
- Two validators with equal voting power
- Default parameters
- Validator Commission: 10%

```
1. Community Tax (20%)
   - 1000 * 0.20 = 200 tokens

2. Separate Pool (5%)
   - 1000 * 0.05 = 50 tokens

3. Staking Rewards (750 tokens)
   Each validator gets 375 tokens (equal voting power)

   Validator 1 (375 tokens):
   NFT Stakers (281.25 tokens):
   - Commission: 281.25 * 0.10 = 28.125 tokens
   - Delegators: 281.25 - 28.125 = 253.125 tokens

   Native Stakers (93.75 tokens):
   - Commission: 93.75 * 0.10 = 9.375 tokens
   - Delegators: 93.75 - 9.375 = 84.375 tokens

   Validator 2 (375 tokens):
   NFT Stakers (281.25 tokens):
   - Commission: 281.25 * 0.10 = 28.125 tokens
   - Delegators: 281.25 - 28.125 = 253.125 tokens

   Native Stakers (93.75 tokens):
   - Commission: 93.75 * 0.10 = 9.375 tokens
   - Delegators: 93.75 - 9.375 = 84.375 tokens
```

Final Distribution:

- Community Pool: 200 tokens (20%)
- Separate Pool: 50 tokens (5%)
- Validator 1 Commission: 37.5 tokens (3.75%)
- Validator 1 NFT Delegators: 253.125 tokens (25.3125%)
- Validator 1 Native Delegators: 84.375 tokens (8.4375%)
- Validator 2 Commission: 37.5 tokens (3.75%)
- Validator 2 NFT Delegators: 253.125 tokens (25.3125%)
- Validator 2 Native Delegators: 84.375 tokens (8.4375%)

## Example 6: Unequal Voting Power

Assume:

- Total Rewards: 1000 tokens
- Validator A: 70% voting power
- Validator B: 30% voting power
- Default parameters
- Validator Commission: 10%

```
1. Community Tax (20%)
   - 1000 * 0.20 = 200 tokens

2. Separate Pool (5%)
   - 1000 * 0.05 = 50 tokens

3. Staking Rewards (750 tokens)
   Validator A (525 tokens = 70% of 750):
   NFT Stakers (393.75 tokens):
   - Commission: 393.75 * 0.10 = 39.375 tokens
   - Delegators: 393.75 - 39.375 = 354.375 tokens

   Native Stakers (131.25 tokens):
   - Commission: 131.25 * 0.10 = 13.125 tokens
   - Delegators: 131.25 - 13.125 = 118.125 tokens

   Validator B (225 tokens = 30% of 750):
   NFT Stakers (168.75 tokens):
   - Commission: 168.75 * 0.10 = 16.875 tokens
   - Delegators: 168.75 - 16.875 = 151.875 tokens

   Native Stakers (56.25 tokens):
   - Commission: 56.25 * 0.10 = 5.625 tokens
   - Delegators: 56.25 - 5.625 = 50.625 tokens
```

Final Distribution:

- Community Pool: 200 tokens (20%)
- Separate Pool: 50 tokens (5%)
- Validator A Commission: 52.5 tokens (5.25%)
- Validator A NFT Delegators: 354.375 tokens (35.4375%)
- Validator A Native Delegators: 118.125 tokens (11.8125%)
- Validator B Commission: 22.5 tokens (2.25%)
- Validator B NFT Delegators: 151.875 tokens (15.1875%)
- Validator B Native Delegators: 50.625 tokens (5.0625%)

These examples demonstrate how rewards flow through the system under various scenarios and configurations. The actual implementation handles these calculations automatically based on the configured parameters and validator states.
