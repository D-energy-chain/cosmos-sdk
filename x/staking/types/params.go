package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Staking params default values
const (
	// DefaultUnbondingTime reflects three weeks in seconds as the default
	// unbonding time.
	// TODO: Justify our choice of default here.
	// Note: DefaultUnbondingTime is not required for DEnergy as the Unbonding is Epoch based.
	DefaultUnbondingTime time.Duration = time.Hour * 24 * 7 * 3

	// Default maximum number of bonded validators
	DefaultMaxValidators uint32 = 100

	// Default maximum entries in a UBD/RED pair
	DefaultMaxEntries uint32 = 7

	// DefaultHistorical entries is 10000. Apps that don't use IBC can ignore this
	// value by not adding the staking module to the application module manager's
	// SetOrderBeginBlockers.
	DefaultHistoricalEntries uint32 = 10000
)

// DefaultMinCommissionRate is set to 0%
var DefaultMinCommissionRate = math.LegacyZeroDec()

// DefaultAllowedValidators is an empty list, meaning all validators are allowed by default
var DefaultAllowedValidators = []string{}

// NewParams creates a new Params instance
func NewParams(unbondingTime time.Duration, maxValidators, maxEntries, historicalEntries uint32, bondDenom string, minCommissionRate math.LegacyDec, allowedValidators []string) Params {
	return Params{
		UnbondingTime:     unbondingTime,
		MaxValidators:     maxValidators,
		MaxEntries:        maxEntries,
		HistoricalEntries: historicalEntries,
		BondDenom:         bondDenom,
		MinCommissionRate: minCommissionRate,
		AllowedValidators: allowedValidators,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultUnbondingTime,
		DefaultMaxValidators,
		DefaultMaxEntries,
		DefaultHistoricalEntries,
		sdk.DefaultBondDenom,
		DefaultMinCommissionRate,
		DefaultAllowedValidators,
	)
}

// unmarshal the current staking params value from store key or panic
func MustUnmarshalParams(cdc *codec.LegacyAmino, value []byte) Params {
	params, err := UnmarshalParams(cdc, value)
	if err != nil {
		panic(err)
	}

	return params
}

// unmarshal the current staking params value from store key
func UnmarshalParams(cdc *codec.LegacyAmino, value []byte) (params Params, err error) {
	err = cdc.Unmarshal(value, &params)
	if err != nil {
		return
	}

	// Ensure AllowedValidators is never nil
	if params.AllowedValidators == nil {
		params.AllowedValidators = []string{}
	}

	return
}

// UnmarshalParamsBinary unmarshals params using binary codec and ensures AllowedValidators is never nil
func UnmarshalParamsBinary(cdc codec.BinaryCodec, value []byte) (params Params, err error) {
	err = cdc.Unmarshal(value, &params)
	if err != nil {
		return params, err
	}

	// Ensure AllowedValidators is never nil
	if params.AllowedValidators == nil {
		params.AllowedValidators = []string{}
	}

	return params, nil
}

// validate a set of params
func (p Params) Validate() error {
	if err := validateUnbondingTime(p.UnbondingTime); err != nil {
		return err
	}

	if err := validateMaxValidators(p.MaxValidators); err != nil {
		return err
	}

	if err := validateMaxEntries(p.MaxEntries); err != nil {
		return err
	}

	if err := validateBondDenom(p.BondDenom); err != nil {
		return err
	}

	if err := validateMinCommissionRate(p.MinCommissionRate); err != nil {
		return err
	}

	if err := validateHistoricalEntries(p.HistoricalEntries); err != nil {
		return err
	}

	if err := validateAllowedValidators(p.AllowedValidators); err != nil {
		return err
	}

	return nil
}

// IsValidatorAllowed checks if a validator address is allowed to create a validator
func (p Params) IsValidatorAllowed(validatorAddr string) bool {
	// If no validators are specified in the allowlist, all validators are allowed
	if len(p.AllowedValidators) == 0 {
		return true
	}

	// Check if the validator is in the allowed list
	for _, allowedAddr := range p.AllowedValidators {
		if allowedAddr == validatorAddr {
			return true
		}
	}

	return false
}

func validateUnbondingTime(i interface{}) error {
	v, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("unbonding time must be positive: %d", v)
	}

	return nil
}

func validateMaxValidators(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("max validators must be positive: %d", v)
	}

	return nil
}

func validateMaxEntries(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("max entries must be positive: %d", v)
	}

	return nil
}

func validateHistoricalEntries(i interface{}) error {
	_, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateBondDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if strings.TrimSpace(v) == "" {
		return errors.New("bond denom cannot be blank")
	}

	if err := sdk.ValidateDenom(v); err != nil {
		return err
	}

	return nil
}

func ValidatePowerReduction(i interface{}) error {
	v, ok := i.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.LT(math.NewInt(1)) {
		return fmt.Errorf("power reduction cannot be lower than 1")
	}

	return nil
}

func validateMinCommissionRate(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("minimum commission rate cannot be nil: %s", v)
	}
	if v.IsNegative() {
		return fmt.Errorf("minimum commission rate cannot be negative: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("minimum commission rate cannot be greater than 100%%: %s", v)
	}

	return nil
}

func validateAllowedValidators(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// Empty list is valid (means all validators are allowed)
	if len(v) == 0 {
		return nil
	}

	// Validate each validator address
	for _, addr := range v {
		if strings.TrimSpace(addr) == "" {
			return errors.New("allowed validator address cannot be blank")
		}
		// Additional validation can be added here for address format if needed
	}

	return nil
}
