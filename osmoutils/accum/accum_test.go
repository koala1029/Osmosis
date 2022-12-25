package accum_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	iavlstore "github.com/cosmos/cosmos-sdk/store/iavl"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/iavl"
	"github.com/stretchr/testify/suite"
	dbm "github.com/tendermint/tm-db"

	"github.com/osmosis-labs/osmosis/osmoutils/accum"
)

type AccumTestSuite struct {
	suite.Suite

	store store.KVStore
}

var (
	testAddressOne   = sdk.AccAddress([]byte("addr1_______________"))
	testAddressTwo   = sdk.AccAddress([]byte("addr2_______________"))
	testAddressThree = sdk.AccAddress([]byte("addr3_______________"))

	emptyPositionOptions = accum.PositionOptions{}
	testNameOne          = "myaccumone"
	testNameTwo          = "myaccumtwo"
	testNameThree        = "myaccumthree"
	denomOne             = "denomone"
	denomTwo             = "denomtwo"

	emptyCoins = sdk.DecCoins(nil)

	initialValueOne      = sdk.MustNewDecFromStr("100.1")
	initialCoinDenomOne  = sdk.NewDecCoinFromDec(denomOne, initialValueOne)
	initialCoinsDenomOne = sdk.NewDecCoins(initialCoinDenomOne)

	positionOne = accum.Record{
		NumShares:        sdk.NewDec(100),
		InitAccumValue:   emptyCoins,
		UnclaimedRewards: emptyCoins,
	}

	positionOneV2 = accum.Record{
		NumShares:        sdk.NewDec(150),
		InitAccumValue:   emptyCoins,
		UnclaimedRewards: emptyCoins,
	}

	positionTwo = accum.Record{
		NumShares:        sdk.NewDec(200),
		InitAccumValue:   emptyCoins,
		UnclaimedRewards: emptyCoins,
	}

	positionThree = accum.Record{
		NumShares:        sdk.NewDec(300),
		InitAccumValue:   emptyCoins,
		UnclaimedRewards: emptyCoins,
	}
)

func withInitialAccumValue(record accum.Record, initialAccum sdk.DecCoins) accum.Record {
	record.InitAccumValue = initialAccum
	return record
}

func withUnclaimedRewards(record accum.Record, unclaimedRewards sdk.DecCoins) accum.Record {
	record.UnclaimedRewards = unclaimedRewards
	return record
}

// Sets/resets KVStore to use for tests under `suite.store`
func (suite *AccumTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	tree, err := iavl.NewMutableTree(db, 100, false)
	suite.Require().NoError(err)
	_, _, err = tree.SaveVersion()
	suite.Require().Nil(err)
	kvstore := iavlstore.UnsafeNewStore(tree)
	suite.store = kvstore
}

func TestTreeTestSuite(t *testing.T) {
	suite.Run(t, new(AccumTestSuite))
}

func (suite *AccumTestSuite) TestMakeAndGetAccum() {
	// We set up store once at beginning so we can test duplicates
	suite.SetupTest()

	type testcase struct {
		accumName  string
		expAccum   accum.AccumulatorObject
		expSetPass bool
		expGetPass bool
	}

	tests := map[string]testcase{
		"create valid accumulator": {
			accumName:  "fee-accumulator",
			expSetPass: true,
			expGetPass: true,
		},
		"create duplicate accumulator": {
			accumName:  "fee-accumulator",
			expSetPass: false,
			expGetPass: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		suite.Run(name, func() {
			// Creates raw accumulator object with test case's accum name and zero initial value
			expAccum := accum.CreateRawAccumObject(suite.store, tc.accumName, sdk.DecCoins(nil))

			err := accum.MakeAccumulator(suite.store, tc.accumName)

			if !tc.expSetPass {
				suite.Require().Error(err)
			}

			actualAccum, err := accum.GetAccumulator(suite.store, tc.accumName)

			if tc.expGetPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expAccum, actualAccum)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AccumTestSuite) TestNewPosition() {
	// We setup store and accum
	// once at beginning so we can test duplicate positions
	suite.SetupTest()

	// Setup.
	accObject := accum.CreateRawAccumObject(suite.store, testNameOne, emptyCoins)

	tests := map[string]struct {
		accObject        accum.AccumulatorObject
		addr             sdk.AccAddress
		numShareUnits    sdk.Dec
		options          accum.PositionOptions
		expectedPosition accum.Record
	}{
		"test address one - position created": {
			accObject:        accObject,
			addr:             testAddressOne,
			numShareUnits:    positionOne.NumShares,
			options:          emptyPositionOptions,
			expectedPosition: positionOne,
		},
		"test address two - position created": {
			accObject:        accObject,
			addr:             testAddressTwo,
			numShareUnits:    positionTwo.NumShares,
			options:          emptyPositionOptions,
			expectedPosition: positionTwo,
		},
		"test address one - position overwritten": {
			accObject:        accObject,
			addr:             testAddressOne,
			numShareUnits:    positionOneV2.NumShares,
			options:          emptyPositionOptions,
			expectedPosition: positionOneV2,
		},
		"test address three - added": {
			accObject:        accObject,
			addr:             testAddressThree,
			numShareUnits:    positionThree.NumShares,
			options:          emptyPositionOptions,
			expectedPosition: positionThree,
		},
		"test address one with non-empty accumulator - position created": {
			accObject:        accum.CreateRawAccumObject(suite.store, testNameTwo, initialCoinsDenomOne),
			addr:             testAddressOne,
			numShareUnits:    positionOne.NumShares,
			options:          emptyPositionOptions,
			expectedPosition: withInitialAccumValue(positionOne, initialCoinsDenomOne),
		},
	}

	for name, tc := range tests {
		tc := tc
		suite.Run(name, func() {

			// System under test.
			tc.accObject.NewPosition(tc.addr, tc.numShareUnits, tc.options)

			// Assertions.
			positions := tc.accObject.GetPosition(tc.addr)
			suite.Require().Equal(tc.expectedPosition, positions)
		})
	}
}

func (suite *AccumTestSuite) TestClaimRewards() {
	var (
		doubleCoinsDenomOne = sdk.NewDecCoinFromDec(denomOne, initialValueOne.MulInt64(2))

		tripleDenomOneAndTwo = sdk.NewDecCoins(
			sdk.NewDecCoinFromDec(denomOne, initialValueOne),
			sdk.NewDecCoinFromDec(denomTwo, sdk.OneDec())).MulDec(sdk.NewDec(3))
	)

	// We setup store and accum
	// once at beginning so we can test duplicate positions
	suite.SetupTest()

	// Setup.

	// 1. No rewards, 2 position accumulator.
	accumNoRewards := accum.CreateRawAccumObject(suite.store, testNameOne, emptyCoins)

	// Create positions at testAddressOne and testAddressTwo.
	accumNoRewards.NewPosition(testAddressOne, positionOne.NumShares, emptyPositionOptions)
	accumNoRewards.NewPosition(testAddressTwo, positionTwo.NumShares, emptyPositionOptions)

	// 2. One accumulator reward coin, 1 position accumulator, no unclaimed rewards in position.
	accumOneReward := accum.CreateRawAccumObject(suite.store, testNameTwo, initialCoinsDenomOne)

	// Create position at testAddressThree.
	accumOneReward = accum.WithPosition(accumOneReward, testAddressThree, withInitialAccumValue(positionThree, initialCoinsDenomOne))

	// Double the accumulator value.
	accumOneReward.SetValue(sdk.NewDecCoins(doubleCoinsDenomOne))

	// 3. Multi accumulator rewards, 2 position accumulator, some unclaimed rewards.
	accumThreeRewards := accum.CreateRawAccumObject(suite.store, testNameThree, sdk.NewDecCoins())

	// Create positions at testAddressOne
	// This position has unclaimed rewards set.
	accumThreeRewards = accum.WithPosition(accumThreeRewards, testAddressOne, withUnclaimedRewards(positionOne, initialCoinsDenomOne))

	// Create positions at testAddressThree with no unclaimed rewards.
	accumThreeRewards.NewPosition(testAddressTwo, positionTwo.NumShares, emptyPositionOptions)

	// Triple the accumulator value.
	accumThreeRewards.SetValue(tripleDenomOneAndTwo)

	tests := map[string]struct {
		accObject      accum.AccumulatorObject
		addr           sdk.AccAddress
		expectedResult sdk.DecCoins
		expectError    error
	}{
		"claim at testAddressOne with no rewards - success": {
			accObject:      accumNoRewards,
			addr:           testAddressOne,
			expectedResult: emptyCoins,
		},
		"claim at testAddressTwo with no rewards - success": {
			accObject:      accumNoRewards,
			addr:           testAddressTwo,
			expectedResult: emptyCoins,
		},
		"claim at testAddressTwo with no rewards - error - no position": {
			accObject:   accumNoRewards,
			addr:        testAddressThree,
			expectError: accum.NoPositionError{Address: testAddressThree},
		},
		"claim at testAddressThree with single reward token - success": {
			accObject: accumOneReward,
			addr:      testAddressThree,
			// denomOne: (200.2 - 100.1) * 300 (accum diff * share count) = 30030
			expectedResult: initialCoinsDenomOne.MulDec(positionThree.NumShares),
		},
		"claim at testAddressOne with multiple reward tokens and unclaimed rewards - success": {
			accObject: accumThreeRewards,
			addr:      testAddressOne,
			// denomOne: (300.3 - 0) * 100 (accum diff * share count) + 100.1 (unclaimed rewards) = 30130.1
			// denomTwo: (3 - 0) * 100 (accum diff * share count) = 300
			expectedResult: tripleDenomOneAndTwo.MulDec(positionOne.NumShares).Add(initialCoinDenomOne),
		},
		"claim at testAddressTwi with multiple reward tokens and no unclaimed rewards - success": {
			accObject: accumThreeRewards,
			addr:      testAddressTwo,
			// denomOne: (300.3 - 0) * 200 (accum diff * share count) = 60060.6
			// denomTwo: (3 - 0) * 200  (accum diff * share count) = 600
			expectedResult: sdk.NewDecCoins(
				initialCoinDenomOne,
				sdk.NewDecCoinFromDec(denomTwo, sdk.OneDec()),
			).MulDec(positionTwo.NumShares).MulDec(sdk.NewDec(3)),
		},
	}

	for name, tc := range tests {
		tc := tc
		suite.Run(name, func() {

			// System under test.
			actualResult, err := tc.accObject.ClaimRewards(tc.addr)

			// Assertions.

			if tc.expectError != nil {
				suite.Require().Error(err)
				suite.Require().Equal(tc.expectError, err)
				return
			}

			suite.Require().NoError(err)

			suite.Require().Equal(tc.expectedResult, actualResult)

			finalPosition := tc.accObject.GetPosition(tc.addr)
			suite.Require().NoError(err)

			// Unclaimed rewards are reset.
			suite.Require().True(finalPosition.UnclaimedRewards.IsZero())
		})
	}
}
