package keeper

import (
	"encoding/binary"
	"sort"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

func typedSubaccountKey(owner string, accountType types.AccountType) []byte {
	key := make([]byte, 0, len(owner)+2)
	key = append(key, []byte(owner)...)
	key = append(key, 0)
	key = append(key, byte(accountType))
	return key
}

func nextBusinessSubaccountNumberKey(owner string) []byte {
	return []byte(owner)
}

// GetTypedSubaccount returns the business subaccount mapped to an owner and type.
func (k Keeper) GetTypedSubaccount(ctx sdk.Context, owner string, accountType types.AccountType) (types.SubaccountId, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.TypedSubaccountKeyPrefix))
	value := store.Get(typedSubaccountKey(owner, accountType))
	if value == nil {
		return types.SubaccountId{}, false
	}
	var id types.SubaccountId
	k.cdc.MustUnmarshal(value, &id)
	return id, true
}

// SetTypedSubaccount stores the unique owner and business account type mapping.
func (k Keeper) SetTypedSubaccount(ctx sdk.Context, owner string, accountType types.AccountType, id types.SubaccountId) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.TypedSubaccountKeyPrefix))
	store.Set(typedSubaccountKey(owner, accountType), k.cdc.MustMarshal(&id))
}

// GetNextBusinessSubaccountNumber returns the next number to consider for an owner.
func (k Keeper) GetNextBusinessSubaccountNumber(ctx sdk.Context, owner string) uint32 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.NextBusinessSubaccountNumberKeyPrefix))
	value := store.Get(nextBusinessSubaccountNumberKey(owner))
	if len(value) != 4 {
		return 1
	}
	return binary.BigEndian.Uint32(value)
}

// SetNextBusinessSubaccountNumber stores the next monotonic number for an owner.
func (k Keeper) SetNextBusinessSubaccountNumber(ctx sdk.Context, owner string, number uint32) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.NextBusinessSubaccountNumberKeyPrefix))
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, number)
	store.Set(nextBusinessSubaccountNumberKey(owner), value)
}

// AllocateBusinessSubaccount allocates and persists a unique business subaccount ID.
// The caller must commit the returned ID and mapping in the same cached context as
// its account creation state transition.
func (k Keeper) AllocateBusinessSubaccount(
	ctx sdk.Context,
	owner string,
	accountType types.AccountType,
) (types.SubaccountId, error) {
	if err := types.ValidateBusinessAccountType(accountType); err != nil {
		return types.SubaccountId{}, err
	}
	if _, exists := k.GetTypedSubaccount(ctx, owner, accountType); exists {
		return types.SubaccountId{}, types.ErrAccountTypeAlreadyExists
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SubaccountKeyPrefix))
	number := k.GetNextBusinessSubaccountNumber(ctx, owner)
	if number == 0 {
		number = 1
	}
	for ; number <= types.MaxSubaccountIdNumber; number++ {
		id := types.SubaccountId{Owner: owner, Number: number}
		if store.Has(id.ToStateKey()) {
			continue
		}
		k.SetTypedSubaccount(ctx, owner, accountType, id)
		k.SetNextBusinessSubaccountNumber(ctx, owner, number+1)
		return id, nil
	}
	return types.SubaccountId{}, types.ErrSubaccountNumberExhausted
}

// GetAllTypedSubaccounts returns all persisted owner/type mappings in stable order.
func (k Keeper) GetAllTypedSubaccounts(ctx sdk.Context) []types.TypedSubaccount {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.TypedSubaccountKeyPrefix))
	result := make([]types.TypedSubaccount, 0)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		separator := -1
		for i, b := range key {
			if b == 0 {
				separator = i
				break
			}
		}
		if separator <= 0 || separator+1 >= len(key) {
			continue
		}
		owner := string(key[:separator])
		accountType := types.AccountType(key[separator+1])
		var id types.SubaccountId
		k.cdc.MustUnmarshal(iterator.Value(), &id)
		result = append(result, types.TypedSubaccount{
			Owner: owner, AccountType: accountType, SubaccountId: &id,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Owner != result[j].Owner {
			return result[i].Owner < result[j].Owner
		}
		return result[i].AccountType < result[j].AccountType
	})
	return result
}

// GetAllNextBusinessSubaccountNumbers returns all persisted business cursors.
func (k Keeper) GetAllNextBusinessSubaccountNumbers(ctx sdk.Context) []types.BusinessSubaccountNumberCursor {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.NextBusinessSubaccountNumberKeyPrefix))
	result := make([]types.BusinessSubaccountNumberCursor, 0)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		result = append(result, types.BusinessSubaccountNumberCursor{
			Owner: string(iterator.Key()), NextNumber: binary.BigEndian.Uint32(iterator.Value()),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Owner < result[j].Owner })
	return result
}
