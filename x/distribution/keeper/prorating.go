package keeper

import (
    "context"

    "cosmossdk.io/math"
)

// calculateProRatingFactor calculates what fraction of a period's rewards
// a delegator should receive based on when they joined relative to the
// period's start and end heights.
func (k Keeper) calculateProRatingFactor(
    _ context.Context,
    delegationHeight uint64,
    periodStartHeight uint64,
    periodEndHeight uint64,
) math.LegacyDec {
    // Delegator was active from max(delegationHeight, periodStartHeight) to periodEndHeight
    var activeStart uint64
    if delegationHeight > periodStartHeight {
        activeStart = delegationHeight
    } else {
        activeStart = periodStartHeight
    }

    if periodEndHeight <= activeStart {
        // No active blocks within the period; zero rewards
        return math.LegacyZeroDec()
    }

    activeBlocks := periodEndHeight - activeStart
    totalBlocks := periodEndHeight - periodStartHeight

    if totalBlocks == 0 {
        // Same-block period; grant full rewards to avoid divide-by-zero
        return math.LegacyOneDec()
    }

    return math.LegacyNewDec(int64(activeBlocks)).Quo(math.LegacyNewDec(int64(totalBlocks)))
}


