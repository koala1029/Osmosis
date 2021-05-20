package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/osmosis-labs/osmosis/x/lockup/types"
)

// getLastLockID returns ID used last time
func (k Keeper) getLastLockID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyLastLockID)
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// setLastLockID save ID used by last lock
func (k Keeper) setLastLockID(ctx sdk.Context, ID uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyLastLockID, sdk.Uint64ToBigEndian(ID))
}

// lockStoreKey returns action store key from ID
func lockStoreKey(ID uint64) []byte {
	return combineKeys(types.KeyPrefixPeriodLock, sdk.Uint64ToBigEndian(ID))
}

// getLockRefs get lock IDs specified on the prefix and timestamp key
func (k Keeper) getLockRefs(ctx sdk.Context, key []byte) []uint64 {
	store := ctx.KVStore(k.storeKey)
	lockIDs := []uint64{}
	if store.Has(key) {
		bz := store.Get(key)
		err := json.Unmarshal(bz, &lockIDs)
		if err != nil {
			panic(err)
		}
	}
	return lockIDs
}

// addLockRefByKey append lock ID into an array associated to provided key
func (k Keeper) addLockRefByKey(ctx sdk.Context, key []byte, lockID uint64) error {
	store := ctx.KVStore(k.storeKey)
	lockIDs := k.getLockRefs(ctx, key)
	if findIndex(lockIDs, lockID) > -1 {
		return fmt.Errorf("lock with same ID exist: %d", lockID)
	}
	lockIDs = append(lockIDs, lockID)
	bz, err := json.Marshal(lockIDs)
	if err != nil {
		return err
	}
	store.Set(key, bz)
	return nil
}

// deleteLockRefByKey removes lock ID from an array associated to provided key
func (k Keeper) deleteLockRefByKey(ctx sdk.Context, key []byte, lockID uint64) error {
	var index = -1
	store := ctx.KVStore(k.storeKey)
	lockIDs := k.getLockRefs(ctx, key)
	lockIDs, index = removeValue(lockIDs, lockID)
	if index < 0 {
		return fmt.Errorf("specific lock with ID %d not found", lockID)
	}
	if len(lockIDs) == 0 {
		if store.Has(key) {
			store.Delete(key)
		}
	} else {
		bz, err := json.Marshal(lockIDs)
		if err != nil {
			return err
		}
		store.Set(key, bz)
	}
	return nil
}
