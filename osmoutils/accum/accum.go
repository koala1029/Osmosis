package accum

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/osmosis-labs/osmosis/osmoutils"
)

// We keep this object as a way to interface with the methods, even though
// only the Accumulator inside is stored in state
type AccumulatorObject struct {
	// Store where accumulator is stored
	store store.KVStore

	// Accumulator's name (pulled from AccumulatorContent)
	name string

	// Accumulator's current value (pulled from AccumulatorContent)
	value sdk.DecCoins
}

type PositionOptions struct{}

// Makes a new accumulator at store/accum/{accumName}
// Returns error if already exists / theres some overlapping keys
func MakeAccumulator(accumStore store.KVStore, accumName string) error {
	if accumStore.Has(formatAccumPrefixKey(accumName)) {
		return errors.New("Accumulator with given name already exists in store")
	}

	// New accumulator values start out at zero
	// TODO: consider whether this should be a parameter instead of always zero
	initAccumValue := sdk.NewDecCoins()

	newAccum := AccumulatorObject{accumStore, accumName, initAccumValue}

	// Stores accumulator in state
	setAccumulator(newAccum, initAccumValue)

	return nil
}

// Gets the current value of the accumulator corresponding to accumName in accumStore
func GetAccumulator(accumStore store.KVStore, accumName string) (AccumulatorObject, error) {
	accumContent := AccumulatorContent{}
	found, err := osmoutils.Get(accumStore, formatAccumPrefixKey(accumName), &accumContent)
	if err != nil {
		return AccumulatorObject{}, err
	}
	if !found {
		return AccumulatorObject{}, errors.New(fmt.Sprintf("Accumulator name %s does not exist in store", accumName))
	}

	accum := AccumulatorObject{accumStore, accumName, accumContent.AccumValue}

	return accum, nil
}

func setAccumulator(accum AccumulatorObject, amt sdk.DecCoins) {
	newAccum := AccumulatorContent{amt}
	osmoutils.MustSet(accum.store, formatAccumPrefixKey(accum.name), &newAccum)
}

// TODO: consider making this increment the accumulator's value instead of overwriting it
// Note: accum receiver is not mutated, only the store representation is.
func (accum AccumulatorObject) UpdateAccumulator(amt sdk.DecCoins) {
	setAccumulator(accum, amt)
}

// NewPosition creates a new position for the given address, with the given number of share units
// It takes a snapshot of the current accumulator value, and sets the position's initial value to that.
// The position is initialized with empty unclaimed rewards
// If there is an existing position for the given address, it is overwritten.
func (accum AccumulatorObject) NewPosition(addr sdk.AccAddress, numShareUnits sdk.Dec, options PositionOptions) {
	createNewPosition(accum, addr, numShareUnits, sdk.NewDecCoins(), options)
}

// AddToPosition adds newShares of shares to addr's position.
// This is functionally equivalent to claiming rewards, closing down the position, and
// opening a fresh one with the new number of shares. We can represent this behavior by
// claiming rewards and moving up the accumulator start value to its current value.
//
// An alternative approach is to simply generate an additional position every time an
// address adds to its position. We do not pursue this path because we want to ensure
// that withdrawal and claiming functions remain constant time and do not scale with the
// number of times a user has added to their position.
//
// Returns nil on success. Returns error when:
// - newShares are negative or zero.
// - there is no existing position at the given address
// - other internal or database error occurs.
func (accum AccumulatorObject) AddToPosition(addr sdk.AccAddress, newShares sdk.Dec) error {
	if !newShares.IsPositive() {
		return errors.New("Attempted to add a non-zero and non-negative number of shares to a position")
	}

	// Get addr's current position
	position, err := getPosition(accum, addr)
	if err != nil {
		return err
	}

	// Save current number of shares and unclaimed rewards
	unclaimedRewards := getTotalRewards(accum, position)
	oldNumShares, err := accum.GetPositionSize(addr)
	if err != nil {
		return err
	}

	// Update user's position with new number of shares while moving its unaccrued rewards
	// into UnclaimedRewards. Starting accumulator value is moved up to accum'scurrent value
	// TODO: decide how to propagate the knowledge of position options.
	createNewPosition(accum, addr, oldNumShares.Add(newShares), unclaimedRewards, PositionOptions{})

	return nil
}

// RemovePosition removes the specified number of shares from a position. Specifically, it claims
// the unclaimed and newly accrued rewards and returns them alongside the redeemed shares. Then, it
// overwrites the position record with the updated number of shares. Since it accrues rewards, it
// also moves up the position's accumulator value to the current accum val.
func (accum AccumulatorObject) RemoveFromPosition(addr sdk.AccAddress, numSharesToRemove sdk.Dec) error {
	// Cannot remove zero or negative shares
	if numSharesToRemove.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("Attempted to remove no/negative shares (%s)", numSharesToRemove)
	}

	// Get addr's current position
	position, err := getPosition(accum, addr)
	if err != nil {
		return err
	}

	// Ensure not removing more shares than exist
	if numSharesToRemove.GT(position.NumShares) {
		return fmt.Errorf("Attempted to remove more shares  (%s) than exist in the position (%s)", numSharesToRemove, position.NumShares)
	}

	// Save current number of shares and unclaimed rewards
	unclaimedRewards := getTotalRewards(accum, position)
	oldNumShares, err := accum.GetPositionSize(addr)
	if err != nil {
		return err
	}

	// TODO: decide how to propagate the knowledge of position options.
	createNewPosition(accum, addr, oldNumShares.Sub(numSharesToRemove), unclaimedRewards, PositionOptions{})

	return nil
}

// GetPositionSize returns the number of shares the position corresponding to `addr`
// in accumulator `accum` has, or an error if no position exists.
func (accum AccumulatorObject) GetPositionSize(addr sdk.AccAddress) (sdk.Dec, error) {
	position, err := getPosition(accum, addr)
	if err != nil {
		return sdk.Dec{}, err
	}

	return position.NumShares, nil
}

// ClaimRewards claims the rewards for the given address, and returns the amount of rewards claimed.
// Upon claiming the rewards, the position at the current address is reset to have no
// unclaimed rewards. The position's accumulator is also set to the current accumulator value.
// Returns error if no position exists for the given address. Returns error if any
// database errors occur.
func (accum AccumulatorObject) ClaimRewards(addr sdk.AccAddress) (sdk.DecCoins, error) {
	position, err := getPosition(accum, addr)
	if err != nil {
		return sdk.DecCoins{}, NoPositionError{addr}
	}

	totalRewards := getTotalRewards(accum, position)

	// Create a completely new position, with no rewards
	// TODO: decide how to propagate the knowledge of position options.
	// TODO: remove the position from state entirely if numShares = zero
	createNewPosition(accum, addr, position.NumShares, sdk.NewDecCoins(), PositionOptions{})

	return totalRewards, nil
}
