package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

// Params queries params of distribution module
func (k Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := k.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

// ValidatorDistributionInfo query validator's commission and self-delegation rewards
func (k Querier) ValidatorDistributionInfo(ctx context.Context, req *types.QueryValidatorDistributionInfoRequest) (*types.QueryValidatorDistributionInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	// self-delegation rewards
	val, err := k.stakingKeeper.Validator(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, errors.Wrap(types.ErrNoValidatorExists, req.ValidatorAddress)
	}

	delAdr := sdk.AccAddress(valAdr)

	del, err := k.stakingKeeper.Delegation(ctx, delAdr, valAdr)
	if err != nil {
		return nil, err
	}

	if del == nil {
		return nil, types.ErrNoDelegationExists
	}

	endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
	if err != nil {
		return nil, err
	}

	rewards, err := k.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	if err != nil {
		return nil, err
	}

	// validator's commission
	validatorCommission, err := k.GetValidatorAccumulatedCommission(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorDistributionInfoResponse{
		Commission:      validatorCommission.Commission,
		OperatorAddress: delAdr.String(),
		SelfBondRewards: rewards,
	}, nil
}

// ValidatorOutstandingRewards queries rewards of a validator address
func (k Querier) ValidatorOutstandingRewards(ctx context.Context, req *types.QueryValidatorOutstandingRewardsRequest) (*types.QueryValidatorOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	validator, err := k.stakingKeeper.Validator(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	if validator == nil {
		return nil, errors.Wrapf(types.ErrNoValidatorExists, req.ValidatorAddress)
	}

	rewards, err := k.GetValidatorOutstandingRewards(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorOutstandingRewardsResponse{Rewards: rewards}, nil
}

// ValidatorCommission queries accumulated commission for a validator
func (k Querier) ValidatorCommission(ctx context.Context, req *types.QueryValidatorCommissionRequest) (*types.QueryValidatorCommissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	validator, err := k.stakingKeeper.Validator(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	if validator == nil {
		return nil, errors.Wrapf(types.ErrNoValidatorExists, req.ValidatorAddress)
	}
	commission, err := k.GetValidatorAccumulatedCommission(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorCommissionResponse{Commission: commission}, nil
}

// ValidatorSlashes queries slash events of a validator
func (k Querier) ValidatorSlashes(ctx context.Context, req *types.QueryValidatorSlashesRequest) (*types.QueryValidatorSlashesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	if req.EndingHeight < req.StartingHeight {
		return nil, status.Errorf(codes.InvalidArgument, "starting height greater than ending height (%d > %d)", req.StartingHeight, req.EndingHeight)
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid validator address")
	}

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	slashesStore := prefix.NewStore(store, types.GetValidatorSlashEventPrefix(valAddr))

	events, pageRes, err := query.GenericFilteredPaginate(k.cdc, slashesStore, req.Pagination, func(key []byte, result *types.ValidatorSlashEvent) (*types.ValidatorSlashEvent, error) {
		if result.ValidatorPeriod < req.StartingHeight || result.ValidatorPeriod > req.EndingHeight {
			return nil, nil
		}

		return result, nil
	}, func() *types.ValidatorSlashEvent {
		return &types.ValidatorSlashEvent{}
	})
	if err != nil {
		return nil, err
	}

	slashes := []types.ValidatorSlashEvent{}
	for _, event := range events {
		slashes = append(slashes, *event)
	}

	return &types.QueryValidatorSlashesResponse{Slashes: slashes, Pagination: pageRes}, nil
}

// DelegationRewards the total rewards accrued by a delegation (both native and NFT)
func (k Querier) DelegationRewards(ctx context.Context, req *types.QueryDelegationRewardsRequest) (*types.QueryDelegationRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	val, err := k.stakingKeeper.Validator(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, errors.Wrap(types.ErrNoValidatorExists, req.ValidatorAddress)
	}

	delAdr, err := k.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	var totalRewards sdk.DecCoins

	// Calculate native delegation rewards
	del, err := k.stakingKeeper.Delegation(ctx, delAdr, valAdr)
	if err == nil && del != nil {
		endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			return nil, err
		}

		nativeRewards, err := k.CalculateDelegationRewards(ctx, val, del, endingPeriod)
		if err != nil {
			return nil, err
		}
		totalRewards = totalRewards.Add(nativeRewards...)
	}

	// Calculate NFT delegation rewards
	hasInfo, err := k.HasDelegatorStartingInfo(ctx, valAdr, delAdr)

	if err == nil && hasInfo {
		// Get delegator starting info to get the starting period and NFT stake
		startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAdr, delAdr)
		if err == nil && !startingInfo.NftStake.IsZero() {
			// End current period and calculate NFT rewards
			endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
			if err != nil {
				return nil, err
			}

			startingPeriod := startingInfo.PreviousPeriod
			nftStake := startingInfo.NftStake

			// Calculate NFT delegation rewards
			nftRewards, err := k.calculateNFTDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, nftStake)
			if err != nil {
				return nil, err
			}

			totalRewards = totalRewards.Add(nftRewards...)
		}
	}

	// Return error if no delegations exist at all
	if del == nil && totalRewards.IsZero() {
		return nil, types.ErrNoDelegationExists
	}

	return &types.QueryDelegationRewardsResponse{Rewards: totalRewards}, nil
}

// DelegationTotalRewards the total rewards accrued by a each validator (both native and NFT)
func (k Querier) DelegationTotalRewards(ctx context.Context, req *types.QueryDelegationTotalRewardsRequest) (*types.QueryDelegationTotalRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	total := sdk.DecCoins{}
	var delRewards []types.DelegationDelegatorReward
	validatorRewardsMap := make(map[string]sdk.DecCoins) // Track rewards per validator

	delAdr, err := k.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	// First, iterate through native delegations
	err = k.stakingKeeper.IterateDelegations(
		ctx, delAdr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
			if err != nil {
				panic(err)
			}

			val, err := k.stakingKeeper.Validator(ctx, valAddr)
			if err != nil {
				panic(err)
			}

			endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
			if err != nil {
				panic(err)
			}

			delReward, err := k.CalculateDelegationRewards(ctx, val, del, endingPeriod)
			if err != nil {
				panic(err)
			}

			validatorRewardsMap[del.GetValidatorAddr()] = delReward
			total = total.Add(delReward...)
			return false
		},
	)
	if err != nil {
		return nil, err
	}

	// Then, check all validators for NFT delegations
	// We need to iterate through all validators to find NFT delegations
	err = k.stakingKeeper.IterateValidators(ctx, func(_ int64, validator stakingtypes.ValidatorI) (stop bool) {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
		if err != nil {
			return false // Continue with other validators
		}

		// Check if delegator starting info exists for NFT delegations
		hasInfo, err := k.HasDelegatorStartingInfo(ctx, valAddr, delAdr)
		if err != nil || !hasInfo {
			return false // Continue with other validators
		}

		// End current period and calculate NFT rewards
		endingPeriod, err := k.IncrementValidatorPeriod(ctx, validator)
		if err != nil {
			return false // Continue with other validators
		}

		// Get delegator starting info to get the starting period and NFT stake
		startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAddr, delAdr)
		if err != nil || startingInfo.NftStake.IsZero() {
			return false // Continue with other validators - no NFT stake
		}

		startingPeriod := startingInfo.PreviousPeriod
		nftStake := startingInfo.NftStake

		// Calculate NFT delegation rewards
		nftRewards, err := k.calculateNFTDelegationRewardsBetween(ctx, validator, startingPeriod, endingPeriod, nftStake)
		if err != nil {
			return false // Continue with other validators
		}

		// Add NFT rewards to existing rewards for this validator
		if existingRewards, exists := validatorRewardsMap[validator.GetOperator()]; exists {
			validatorRewardsMap[validator.GetOperator()] = existingRewards.Add(nftRewards...)
		} else {
			validatorRewardsMap[validator.GetOperator()] = nftRewards
		}
		total = total.Add(nftRewards...)

		return false // Continue with other validators
	})
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	for valAddr, rewards := range validatorRewardsMap {
		delRewards = append(delRewards, types.NewDelegationDelegatorReward(valAddr, rewards))
	}

	return &types.QueryDelegationTotalRewardsResponse{Rewards: delRewards, Total: total}, nil
}

// DelegatorValidators queries the validators list of a delegator
func (k Querier) DelegatorValidators(ctx context.Context, req *types.QueryDelegatorValidatorsRequest) (*types.QueryDelegatorValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	delAdr, err := k.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}
	var validators []string

	err = k.stakingKeeper.IterateDelegations(
		ctx, delAdr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			validators = append(validators, del.GetValidatorAddr())
			return false
		},
	)

	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorValidatorsResponse{Validators: validators}, nil
}

// DelegatorWithdrawAddress queries Query/delegatorWithdrawAddress
func (k Querier) DelegatorWithdrawAddress(ctx context.Context, req *types.QueryDelegatorWithdrawAddressRequest) (*types.QueryDelegatorWithdrawAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}
	delAdr, err := k.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAdr)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorWithdrawAddressResponse{WithdrawAddress: withdrawAddr.String()}, nil
}

// CommunityPool queries the community pool coins
func (k Querier) CommunityPool(ctx context.Context, req *types.QueryCommunityPoolRequest) (*types.QueryCommunityPoolResponse, error) {
	pool, err := k.FeePool.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCommunityPoolResponse{Pool: pool.CommunityPool}, nil
}

// ValidatorEpochPerformance queries a validator's performance for a specific epoch
func (k Querier) ValidatorEpochPerformance(ctx context.Context, req *types.QueryValidatorEpochPerformanceRequest) (*types.QueryValidatorEpochPerformanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	if req.EpochIdentifier == "" {
		return nil, status.Error(codes.InvalidArgument, "empty epoch identifier")
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	performance, err := k.GetValidatorEpochPerformance(ctx, valAddr, req.EpochIdentifier, req.EpochNumber)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryValidatorEpochPerformanceResponse{Performance: performance}, nil
}

// ValidatorEpochPerformances queries all epoch performances for a validator
func (k Querier) ValidatorEpochPerformances(ctx context.Context, req *types.QueryValidatorEpochPerformancesRequest) (*types.QueryValidatorEpochPerformancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var performances []types.ValidatorEpochPerformance
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefixStore := prefix.NewStore(store, types.GetValidatorEpochPerformancePrefix(valAddr))

	pageRes, err := query.Paginate(prefixStore, req.Pagination, func(key, value []byte) error {
		var performance types.ValidatorEpochPerformance
		if err := k.cdc.Unmarshal(value, &performance); err != nil {
			return err
		}
		performances = append(performances, performance)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryValidatorEpochPerformancesResponse{
		Performances: performances,
		Pagination:   pageRes,
	}, nil
}

// EpochPerformances queries all validator performances for a specific epoch
func (k Querier) EpochPerformances(ctx context.Context, req *types.QueryEpochPerformancesRequest) (*types.QueryEpochPerformancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.EpochIdentifier == "" {
		return nil, status.Error(codes.InvalidArgument, "empty epoch identifier")
	}

	// Use the more efficient method from keeper
	performances, err := k.GetAllValidatorEpochPerformanceForEpoch(ctx, req.EpochIdentifier, req.EpochNumber)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Simple in-memory pagination since we already have all the data
	limit := uint64(100) // default limit
	if req.Pagination != nil && req.Pagination.Limit != 0 {
		limit = req.Pagination.Limit
	}

	var offset uint64
	if req.Pagination != nil && req.Pagination.Offset != 0 {
		offset = req.Pagination.Offset
	}

	total := uint64(len(performances))

	// Handle offset bounds
	if offset >= total {
		return &types.QueryEpochPerformancesResponse{
			Performances: []types.ValidatorEpochPerformance{},
			Pagination: &query.PageResponse{
				NextKey: nil,
				Total:   total,
			},
		}, nil
	}

	// Calculate end index
	end := offset + limit
	if end > total {
		end = total
	}

	paginatedPerformances := performances[offset:end]

	// Set next key if there are more results
	var nextKey []byte
	if end < total {
		nextKey = sdk.Uint64ToBigEndian(end)
	}

	return &types.QueryEpochPerformancesResponse{
		Performances: paginatedPerformances,
		Pagination: &query.PageResponse{
			NextKey: nextKey,
			Total:   total,
		},
	}, nil
}
