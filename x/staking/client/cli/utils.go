package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// validator struct to define the fields of the validator
type validator struct {
	Amount               sdk.Coin
	PubKey               cryptotypes.PubKey
	Moniker              string
	Identity             string
	Website              string
	Security             string
	Details              string
	CommissionRates      types.CommissionRates
	MinSelfDelegation    math.Int
	NftContractAddress   string
	TokenId              math.Int
	NftAmount            math.Int
	MinNftSelfDelegation math.Int
	Watts                uint64
	Country              string
}

func parseAndValidateValidatorJSON(cdc codec.Codec, path string) (validator, error) {
	type internalVal struct {
		Amount               string          `json:"amount"`
		PubKey               json.RawMessage `json:"pubkey"`
		Moniker              string          `json:"moniker"`
		Identity             string          `json:"identity,omitempty"`
		Website              string          `json:"website,omitempty"`
		Security             string          `json:"security,omitempty"`
		Details              string          `json:"details,omitempty"`
		CommissionRate       string          `json:"commission-rate"`
		CommissionMaxRate    string          `json:"commission-max-rate"`
		CommissionMaxChange  string          `json:"commission-max-change-rate"`
		MinSelfDelegation    string          `json:"min-self-delegation"`
		NftContractAddress   string          `json:"nft-contract"`
		TokenId              string          `json:"token-id"`
		NftAmount            string          `json:"nft-amount"`
		MinNftSelfDelegation string          `json:"min-nft-self-delegation"`
		Watts                string          `json:"watts"`
		Country              string          `json:"country"`
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return validator{}, err
	}

	var v internalVal
	err = json.Unmarshal(contents, &v)
	if err != nil {
		return validator{}, err
	}

	// Validate Website URL if provided
	if v.Website != "" {
		if _, err := url.ParseRequestURI(v.Website); err != nil {
			return validator{}, fmt.Errorf("invalid website URL: %w", err)
		}
	}

	// Validate Security contact if provided (should be an email or URL)
	if v.Security != "" {
		if _, err := url.ParseRequestURI(v.Security); err != nil {
			// If not a URL, check if it's a valid email format
			if !strings.Contains(v.Security, "@") {
				return validator{}, fmt.Errorf("security contact must be a valid URL or email address")
			}
		}
	}

	if v.Amount == "" {
		return validator{}, fmt.Errorf("must specify amount of coins to bond")
	}
	amount, err := sdk.ParseCoinNormalized(v.Amount)
	if err != nil {
		return validator{}, err
	}

	if v.PubKey == nil {
		return validator{}, fmt.Errorf("must specify the JSON encoded pubkey")
	}
	var pk cryptotypes.PubKey
	if err := cdc.UnmarshalInterfaceJSON(v.PubKey, &pk); err != nil {
		return validator{}, err
	}

	if v.Moniker == "" {
		return validator{}, fmt.Errorf("must specify the moniker name")
	}

	commissionRates, err := buildCommissionRates(v.CommissionRate, v.CommissionMaxRate, v.CommissionMaxChange)
	if err != nil {
		return validator{}, err
	}

	// Validate commission rates
	if err := commissionRates.Validate(); err != nil {
		return validator{}, err
	}

	if v.MinSelfDelegation == "" {
		return validator{}, fmt.Errorf("must specify minimum self delegation")
	}
	minSelfDelegation, ok := math.NewIntFromString(v.MinSelfDelegation)

	if !ok || !minSelfDelegation.IsPositive() {
		return validator{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "minimum self delegation must be a positive integer")
	}

	if v.NftContractAddress == "" {
		return validator{}, fmt.Errorf("must specify the nft contract address")
	}

	if v.TokenId == "" {
		return validator{}, fmt.Errorf("must specify nft token-id")
	}

	tokenId, ok := math.NewIntFromString(v.TokenId)
	if !ok || !tokenId.IsPositive() {
		return validator{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "token id must be a positive integer")
	}

	if v.NftAmount == "" {
		return validator{}, fmt.Errorf("must specify NFT amount to be staked")
	}

	nftAmount, ok := math.NewIntFromString(v.NftAmount)
	if !ok || !nftAmount.IsPositive() {
		return validator{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "nft amount must be a positive integer")
	}

	if v.MinNftSelfDelegation == "" {
		return validator{}, fmt.Errorf("must specify minimum NFT self delegation")
	}

	minNftSelfDelegation, ok := math.NewIntFromString(v.MinNftSelfDelegation)
	if !ok || !minNftSelfDelegation.IsPositive() {
		return validator{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "minimum NFT self delegation must be a positive integer")
	}

	// Validate Watts (should be within reasonable range, e.g., 1-100000)
	watts, err := strconv.ParseUint(v.Watts, 10, 64)
	if err != nil {
		return validator{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "watts must be a positive integer")
	}
	if watts < 1 || watts > 100000 {
		return validator{}, fmt.Errorf("watts must be between 1 and 100000")
	}

	// Validate Country code (should be a valid ISO 3166-1 alpha-3 code)
	if v.Country == "" {
		return validator{}, fmt.Errorf("must specify the country of the node")
	}

	if len(v.Country) != 3 {
		return validator{}, fmt.Errorf("country code must be a valid ISO 3166-1 alpha-3 code")
	}

	// Validate ISO 3166-1 alpha-3 country code
	countryCode := CountryByName(v.Country)
	if countryCode == Unknown {
		return validator{}, fmt.Errorf("invalid ISO 3166-1 alpha-3 country code: %s", v.Country)
	}
	v.Country = countryCode.Alpha3() // Normalize to uppercase alpha-3 code

	return validator{
		Amount:               amount,
		PubKey:               pk,
		Moniker:              v.Moniker,
		Identity:             v.Identity,
		Website:              v.Website,
		Security:             v.Security,
		Details:              v.Details,
		CommissionRates:      commissionRates,
		MinSelfDelegation:    minSelfDelegation,
		NftContractAddress:   v.NftContractAddress,
		TokenId:              tokenId,
		NftAmount:            nftAmount,
		MinNftSelfDelegation: minNftSelfDelegation,
		Watts:                watts,
		Country:              v.Country,
	}, nil
}

func buildCommissionRates(rateStr, maxRateStr, maxChangeRateStr string) (commission types.CommissionRates, err error) {
	if rateStr == "" || maxRateStr == "" || maxChangeRateStr == "" {
		return commission, errors.New("must specify all validator commission parameters")
	}

	rate, err := math.LegacyNewDecFromStr(rateStr)
	if err != nil {
		return commission, err
	}

	maxRate, err := math.LegacyNewDecFromStr(maxRateStr)
	if err != nil {
		return commission, err
	}

	maxChangeRate, err := math.LegacyNewDecFromStr(maxChangeRateStr)
	if err != nil {
		return commission, err
	}

	commission = types.NewCommissionRates(rate, maxRate, maxChangeRate)

	return commission, nil
}
