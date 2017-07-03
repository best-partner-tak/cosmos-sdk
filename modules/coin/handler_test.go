package coin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/basecoin"
	"github.com/tendermint/basecoin/stack"
	"github.com/tendermint/basecoin/types"
)

// this makes sure that txs are rejected with invalid data or permissions
func TestHandlerValidation(t *testing.T) {
	assert := assert.New(t)

	// these are all valid, except for minusCoins
	addr1 := basecoin.Actor{App: "coin", Address: []byte{1, 2}}
	addr2 := basecoin.Actor{App: "role", Address: []byte{7, 8}}
	someCoins := types.Coins{{"atom", 123}}
	doubleCoins := types.Coins{{"atom", 246}}
	minusCoins := types.Coins{{"eth", -34}}

	cases := []struct {
		valid bool
		tx    basecoin.Tx
		perms []basecoin.Actor
	}{
		// auth works with different apps
		{true,
			NewSendTx(
				[]TxInput{NewTxInput(addr1, someCoins, 2)},
				[]TxOutput{NewTxOutput(addr2, someCoins)}),
			[]basecoin.Actor{addr1}},
		{true,
			NewSendTx(
				[]TxInput{NewTxInput(addr2, someCoins, 2)},
				[]TxOutput{NewTxOutput(addr1, someCoins)}),
			[]basecoin.Actor{addr1, addr2}},
		// check multi-input with both sigs
		{true,
			NewSendTx(
				[]TxInput{NewTxInput(addr1, someCoins, 2), NewTxInput(addr2, someCoins, 3)},
				[]TxOutput{NewTxOutput(addr1, doubleCoins)}),
			[]basecoin.Actor{addr1, addr2}},
		// wrong permissions fail
		{false,
			NewSendTx(
				[]TxInput{NewTxInput(addr1, someCoins, 2)},
				[]TxOutput{NewTxOutput(addr2, someCoins)}),
			[]basecoin.Actor{}},
		{false,
			NewSendTx(
				[]TxInput{NewTxInput(addr1, someCoins, 2)},
				[]TxOutput{NewTxOutput(addr2, someCoins)}),
			[]basecoin.Actor{addr2}},
		{false,
			NewSendTx(
				[]TxInput{NewTxInput(addr1, someCoins, 2), NewTxInput(addr2, someCoins, 3)},
				[]TxOutput{NewTxOutput(addr1, doubleCoins)}),
			[]basecoin.Actor{addr1}},
		// invalid input fails
		{false,
			NewSendTx(
				[]TxInput{NewTxInput(addr1, minusCoins, 2)},
				[]TxOutput{NewTxOutput(addr2, minusCoins)}),
			[]basecoin.Actor{addr2}},
	}

	for i, tc := range cases {
		ctx := stack.MockContext("base-chain").WithPermissions(tc.perms...)
		_, err := checkTx(ctx, tc.tx)
		if tc.valid {
			assert.Nil(err, "%d: %+v", i, err)
		} else {
			assert.NotNil(err, "%d", i)
		}
	}
}

func TestDeliverTx(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// some sample settings
	addr1 := basecoin.Actor{App: "coin", Address: []byte{1, 2}}
	addr2 := basecoin.Actor{App: "role", Address: []byte{7, 8}}
	addr3 := basecoin.Actor{App: "coin", Address: []byte{6, 5, 4, 3}}

	someCoins := types.Coins{{"atom", 123}}
	moreCoins := types.Coins{{"atom", 6487}}
	diffCoins := moreCoins.Minus(someCoins)
	otherCoins := types.Coins{{"eth", 11}}
	mixedCoins := someCoins.Plus(otherCoins)

	type money struct {
		addr  basecoin.Actor
		coins types.Coins
	}

	cases := []struct {
		init  []money
		tx    basecoin.Tx
		perms []basecoin.Actor
		final []money // nil for error
	}{
		{
			[]money{{addr1, moreCoins}},
			NewSendTx(
				[]TxInput{NewTxInput(addr1, someCoins, 1)},
				[]TxOutput{NewTxOutput(addr2, someCoins)}),
			[]basecoin.Actor{addr1},
			[]money{{addr1, diffCoins}, {addr2, someCoins}},
		},
		// simple multi-sig 2 accounts to 1
		{
			[]money{{addr1, mixedCoins}, {addr2, moreCoins}},
			NewSendTx(
				[]TxInput{NewTxInput(addr1, otherCoins, 1), NewTxInput(addr2, someCoins, 1)},
				[]TxOutput{NewTxOutput(addr3, mixedCoins)}),
			[]basecoin.Actor{addr1, addr2},
			[]money{{addr1, someCoins}, {addr2, diffCoins}, {addr3, mixedCoins}},
		},
		// multi-sig with one account sending many times
		{
			[]money{{addr1, moreCoins.Plus(otherCoins)}},
			NewSendTx(
				[]TxInput{NewTxInput(addr1, otherCoins, 1), NewTxInput(addr1, someCoins, 2)},
				[]TxOutput{NewTxOutput(addr2, mixedCoins)}),
			[]basecoin.Actor{addr1},
			[]money{{addr1, diffCoins}, {addr2, mixedCoins}},
		},
	}

	h := NewHandler()
	for i, tc := range cases {
		// setup the cases....
		store := types.NewMemKVStore()
		for _, m := range tc.init {
			acct := Account{Coins: m.coins}
			err := storeAccount(store, h.makeKey(m.addr), acct)
			require.Nil(err, "%d: %+v", i, err)
		}

		ctx := stack.MockContext("base-chain").WithPermissions(tc.perms...)
		_, err := h.DeliverTx(ctx, store, tc.tx)
		if len(tc.final) > 0 { // valid
			assert.Nil(err, "%d: %+v", i, err)
			// make sure the final balances are correct
			for _, f := range tc.final {
				acct, err := loadAccount(store, h.makeKey(f.addr))
				assert.Nil(err, "%d: %+v", i, err)
				assert.Equal(f.coins, acct.Coins)
			}
		} else {
			assert.NotNil(err, "%d", i)
			// TODO: make sure balances unchanged!
		}

	}

}
