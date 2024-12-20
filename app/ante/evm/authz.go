package evm

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/common"

	evmtypes "github.com/evmos/evmos/v12/x/evm/types"
)

// EthAuthorizationDecorator takes care of authorization of transactions. If a transaction is executed
// on behalf of another account, it checks if the granter has granted permission to the grantee.
// NOTE: This decorator does not perform any validation
type EthAuthorizationDecorator struct {
	authzKeeper AuthzKeeper
}

// NewEthAuthorizationDecorator creates a new NewEthAuthorizationDecorator
func NewEthAuthorizationDecorator(authzKeeper AuthzKeeper) EthAuthorizationDecorator {
	return EthAuthorizationDecorator{authzKeeper: authzKeeper}
}

// AnteHandle handles the authorization of the transaction.
// If the transaction is executed on behalf of another account, it checks if the granter has granted
// permission to the grantee.
// CONTRACT:
//   - From must be non-empty
//   - Both From and OnBehalf (if present) must be valid ethereum addresses in hex format
//   - Grant must be generic
func (d EthAuthorizationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	var ethMsgTypeURL = sdk.MsgTypeURL(new(evmtypes.MsgEthereumTx))

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		if msgEthTx.OnBehalf == "" {
			// this message is not executed on behalf of another account
			continue
		}

		granter := sdk.AccAddress(common.FromHex(msgEthTx.OnBehalf))
		grantee := sdk.AccAddress(common.FromHex(msgEthTx.From))

		authorization, _ := d.authzKeeper.GetAuthorization(ctx, grantee, granter, ethMsgTypeURL)
		if authorization == nil {
			// Nil is returned under the following circumstances:
			//   - No grant is found.
			//   - A grant is found, but it is expired.
			//   - There was an error getting the authorization from the grant.
			return ctx, errorsmod.Wrapf(errortypes.ErrUnauthorized, "granter has not granted permission to execute MsgEthereumTx on their behalf or it has been expired: granter %s, grantee %s", granter, grantee)
		}

		_, ok = authorization.(*authz.GenericAuthorization)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrLogic, "expected generic authorization, got %T", authorization)
		}

		resp, err := authorization.Accept(ctx, msgEthTx)
		if err != nil {
			// This should never happen, as generic authorization always returns nil error
			return ctx, errorsmod.Wrapf(err, "accept transaction: granter %s, grantee %s", granter, grantee)
		}

		if !resp.Accept {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnauthorized, "granter has not granted permission to execute MsgEthereumTx on their behalf: granter %s, grantee %s", granter, grantee)
		}
	}

	return next(ctx, tx, simulate)
}
