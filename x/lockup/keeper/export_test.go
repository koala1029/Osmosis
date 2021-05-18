package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/osmosis-labs/osmosis/x/lockup/types"
)

func (k Keeper) AddLockRefByKey(ctx sdk.Context, key []byte, lockID uint64) error {
	return k.addLockRefByKey(ctx, key, lockID)
}

func (k Keeper) DeleteLockRefByKey(ctx sdk.Context, key []byte, lockID uint64) error {
	return k.deleteLockRefByKey(ctx, key, lockID)
}

func (k Keeper) GetLockRefs(ctx sdk.Context, key []byte) types.LockIDs {
	return k.getLockRefs(ctx, key)
}
