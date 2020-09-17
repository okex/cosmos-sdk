package mint

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestBeginBlocker(t *testing.T) {
	mintParams := Params{
		MintDenom:      sdk.DefaultBondDenom,
		DeflationRate:  sdk.NewDecWithPrec(50, 2),
		BlocksPerYear:  uint64(30),
		DeflationEpoch: uint64(3),
	}
	var balance int64 = 10000
	mapp, _ := getMockApp(t, 1, balance, mintParams)

	mapp.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: int64(2)}})
	ctx := mapp.BaseApp.NewContext(false, abci.Header{}).WithBlockHeight(int64(2))

	// mint rate test
	minter := mapp.mintKeeper.GetMinterCustom(ctx)
	ratePerBlock0 := minter.MintedPerBlock.AmountOf(sdk.DefaultBondDenom)
	assert.EqualValues(t, ratePerBlock0, mapp.mintKeeper.GetOriginalMintedPerBlock())

	var curHeight int64 = 2
	runBlocks := int64(mintParams.BlocksPerYear * mintParams.DeflationEpoch)
	for ; curHeight < runBlocks; curHeight++ {
		mapp.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: curHeight}})
		mapp.EndBlock(abci.RequestEndBlock{Height: curHeight})
		mapp.Commit()
	}

	// this year mint test
	curSupply0 := mapp.supplyKeeper.GetSupply(ctx).GetTotal().AmountOf(sdk.DefaultBondDenom)
	curCoin0 := curSupply0
	rawCoin := sdk.NewDec(balance)
	assert.EqualValues(t, curCoin0.Sub(rawCoin), ratePerBlock0.Mul(sdk.NewDec(curHeight-1)))

	// next year mint test
	mapp.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: curHeight}})
	ctx = mapp.BaseApp.NewContext(false, abci.Header{}).WithBlockHeight(curHeight)
	curSupply1 := mapp.supplyKeeper.GetSupply(ctx).GetTotal().AmountOf(sdk.DefaultBondDenom)
	curCoin1 := curSupply1

	minter = mapp.mintKeeper.GetMinterCustom(ctx)
	ratePerBlock1 := minter.MintedPerBlock.AmountOf(sdk.DefaultBondDenom)
	assert.EqualValues(t, ratePerBlock1, mintParams.DeflationRate.Mul(mapp.mintKeeper.GetOriginalMintedPerBlock()))

	// annual mint test
	step1Mint := ratePerBlock0.Mul(sdk.NewDec(runBlocks))
	step2Mint := ratePerBlock1.Mul(sdk.NewDec(curHeight - runBlocks - 1))
	totalMint := step1Mint.Add(step2Mint)
	assert.EqualValues(t, curCoin1.Sub(rawCoin), totalMint)
}

func TestMintZero(t *testing.T) {
	mintParams := Params{
		MintDenom:      sdk.DefaultBondDenom,
		DeflationRate:  sdk.NewDecWithPrec(50, 2),
		BlocksPerYear:  uint64(10),
		DeflationEpoch: uint64(3),
	}
	var balance int64 = 10000
	mapp, _ := getMockApp(t, 1, balance, mintParams)

	var curHeight int64 = 2
	for ; curHeight < 700; curHeight++ {
		mapp.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: curHeight}})
		mapp.EndBlock(abci.RequestEndBlock{Height: curHeight})
		mapp.Commit()
	}

	mapp.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: curHeight}})
	ctx := mapp.BaseApp.NewContext(false, abci.Header{}).WithBlockHeight(curHeight)
	minter := mapp.mintKeeper.GetMinterCustom(ctx)
	assert.EqualValues(t, true, minter.MintedPerBlock.AmountOf(mintParams.MintDenom).IsZero())
}