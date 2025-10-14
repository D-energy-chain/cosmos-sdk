package types

import (
	context "context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	AddressCodec() address.Codec
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI
	// TODO remove with genesis 2-phases refactor https://github.com/cosmos/cosmos-sdk/issues/2862
	SetModuleAccount(context.Context, sdk.ModuleAccountI)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins

	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins

	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	BlockedAddr(addr sdk.AccAddress) bool
}

// StakingKeeper expected staking keeper (noalias)
type StakingKeeper interface {
	ValidatorAddressCodec() address.Codec
	ConsensusAddressCodec() address.Codec
	// iterate through validators by operator address, execute func for each validator
	IterateValidators(context.Context,
		func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error

	Validator(context.Context, sdk.ValAddress) (stakingtypes.ValidatorI, error)            // get a particular validator by operator address
	ValidatorByConsAddr(context.Context, sdk.ConsAddress) (stakingtypes.ValidatorI, error) // get a particular validator by consensus address

	// Delegation allows for getting a particular delegation for a given validator
	// and delegator outside the scope of the staking module.
	Delegation(context.Context, sdk.AccAddress, sdk.ValAddress) (stakingtypes.DelegationI, error)
	NFTDelegationShares(context.Context, sdk.AccAddress, sdk.ValAddress) (stakingtypes.NFTDelegationI, error)

	IterateDelegations(ctx context.Context, delegator sdk.AccAddress,
		fn func(index int64, delegation stakingtypes.DelegationI) (stop bool)) error

	GetAllSDKDelegations(ctx context.Context) ([]stakingtypes.Delegation, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	GetAllDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress) ([]stakingtypes.Delegation, error)

	// NFT delegation methods - for calculating NFT delegator rewards
	// GetNFTDelegatorShares returns the total NFT shares for a delegator with a specific validator
	GetNFTDelegatorShares(ctx context.Context, delegator sdk.AccAddress, validator sdk.ValAddress) (math.LegacyDec, error)
	// GetNFTDelegations returns all NFT delegations for a specific delegator with a specific validator
	GetNFTDelegations(ctx context.Context, delegator sdk.AccAddress, validator sdk.ValAddress) ([]*stakingtypes.NFTDelegation, error)
}

// StakingHooks event hooks for staking validator object (noalias)
type StakingHooks interface {
	AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error // Must be called when a validator is created
	AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
}

// NFTStakingHooks event hooks for NFT staking operations
type NFTStakingHooks interface {
	BeforeNFTDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
	BeforeNFTDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
	AfterNFTDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
	BeforeNFTDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error
}

// EpochHooks event hooks for epoch processing
type EpochHooks interface {
	// the first block whose timestamp is after the duration is counted as the end of the epoch
	AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64)
	// new epoch is next block of epoch end block
	BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64)
}

// EpochHooksWrapper is a wrapper for modules to inject EpochHooks using depinject.
type EpochHooksWrapper struct{ EpochHooks }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (EpochHooksWrapper) IsOnePerModuleType() {}
