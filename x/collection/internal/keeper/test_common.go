package keeper

import (
	"github.com/line/lbm-sdk/x/auth"
	"github.com/line/lbm-sdk/x/collection/internal/types"
	"github.com/line/lbm-sdk/x/contract"
	"github.com/line/lbm-sdk/x/params"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	"github.com/line/lbm-sdk/codec"
	"github.com/line/lbm-sdk/store"
	sdk "github.com/line/lbm-sdk/types"
)

func TestKeeper() (sdk.Context, store.CommitMultiStore, Keeper) {
	keyAuth := sdk.NewKVStoreKey(auth.StoreKey)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)
	keyCollection := sdk.NewKVStoreKey(types.StoreKey)
	keyContract := sdk.NewKVStoreKey(contract.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAuth, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyCollection, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyContract, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)

	if err := ms.LoadLatestVersion(); err != nil {
		panic(err)
	}

	cdc := codec.New()
	types.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	cdc.Seal()

	paramsKeeper := params.NewKeeper(cdc, keyParams, tkeyParams)
	authSubspace := paramsKeeper.Subspace(auth.DefaultParamspace)

	// add keepers
	accountKeeper := auth.NewAccountKeeper(cdc, keyAuth, authSubspace, auth.ProtoBaseAccount)
	paramsSpace := paramsKeeper.Subspace(types.DefaultParamspace)
	keeper := NewKeeper(
		cdc,
		accountKeeper,
		contract.NewContractKeeper(cdc, keyContract),
		paramsSpace,
		keyCollection,
	)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "test-chain-id"}, false, log.NewNopLogger())
	keeper.SetParams(ctx, types.DefaultParams())
	return ctx, ms, keeper
}