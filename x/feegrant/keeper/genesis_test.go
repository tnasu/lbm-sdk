package keeper_test

import (
	"testing"

	tmproto "github.com/line/ostracon/proto/ostracon/types"
	"github.com/stretchr/testify/suite"

	"github.com/line/lbm-sdk/x/feegrant/types"

	codectypes "github.com/line/lbm-sdk/codec/types"
	"github.com/line/lbm-sdk/crypto/keys/secp256k1"
	"github.com/line/lbm-sdk/simapp"
	"github.com/line/lbm-sdk/testutil/testdata"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/x/feegrant/keeper"
)

type GenesisTestSuite struct {
	suite.Suite
	ctx    sdk.Context
	keeper keeper.Keeper
}

func (suite *GenesisTestSuite) SetupTest() {
	checkTx := false
	app := simapp.Setup(checkTx)
	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1})
	suite.keeper = app.FeeGrantKeeper
}

var (
	granteePub  = secp256k1.GenPrivKey().PubKey()
	granterPub  = secp256k1.GenPrivKey().PubKey()
	granteeAddr = sdk.BytesToAccAddress(granteePub.Address())
	granterAddr = sdk.BytesToAccAddress(granterPub.Address())
)

func (suite *GenesisTestSuite) TestImportExportGenesis() {
	coins := sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1_000)))
	now := suite.ctx.BlockHeader().Time
	oneYear := now.AddDate(1, 0, 0)
	msgSrvr := keeper.NewMsgServerImpl(suite.keeper)

	allowance := &types.BasicAllowance{SpendLimit: coins, Expiration: &oneYear}
	err := suite.keeper.GrantAllowance(suite.ctx, granterAddr, granteeAddr, allowance)
	suite.Require().NoError(err)

	genesis, err := suite.keeper.ExportGenesis(suite.ctx)
	suite.Require().NoError(err)
	// revoke fee allowance
	_, err = msgSrvr.RevokeAllowance(sdk.WrapSDKContext(suite.ctx), &types.MsgRevokeAllowance{
		Granter: granterAddr.String(),
		Grantee: granteeAddr.String(),
	})
	suite.Require().NoError(err)
	err = suite.keeper.InitGenesis(suite.ctx, genesis)
	suite.Require().NoError(err)

	newGenesis, err := suite.keeper.ExportGenesis(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(genesis, newGenesis)
}

func (suite *GenesisTestSuite) TestInitGenesis() {
	any, err := codectypes.NewAnyWithValue(&testdata.Dog{})
	suite.Require().NoError(err)

	testCases := []struct {
		name          string
		feeAllowances []types.Grant
	}{
		{
			"invalid granter",
			[]types.Grant{
				{
					Granter: "invalid granter",
					Grantee: granteeAddr.String(),
				},
			},
		},
		{
			"invalid grantee",
			[]types.Grant{
				{
					Granter: granterAddr.String(),
					Grantee: "invalid grantee",
				},
			},
		},
		{
			"invalid allowance",
			[]types.Grant{
				{
					Granter:   granterAddr.String(),
					Grantee:   granteeAddr.String(),
					Allowance: any,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			err := suite.keeper.InitGenesis(suite.ctx, &types.GenesisState{Allowances: tc.feeAllowances})
			suite.Require().Error(err)
		})
	}
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}