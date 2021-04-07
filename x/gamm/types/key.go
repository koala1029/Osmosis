package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "gamm"

	StoreKey = ModuleName

	RouterKey = ModuleName

	QuerierRoute = ModuleName
)

var (
	PoolAddressPrefix = []byte("gamm_liquidity_pool")
	GlobalPoolNumber  = []byte("gamm_global_pool_number")
	// Used for querying to paginate the registered pool numbers.
	PaginationPoolNumbers = []byte("gamm_pool_numbers_pagination")
)

func GetPoolShareDenom(poolId uint64) string {
	return fmt.Sprintf("gamm/pool/%d", poolId)
}

func GetKeyPaginationPoolNumbers(poolId uint64) []byte {
	return append(PaginationPoolNumbers, sdk.Uint64ToBigEndian(poolId)...)
}
