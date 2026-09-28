package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// ValidateBasic performs stateless validation for MsgCreateSubaccount.
func (msg *MsgCreateSubaccount) ValidateBasic() error {
	if msg == nil {
		return ErrInvalidAccountType
	}
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSubaccountIdOwner
	}
	return ValidateBusinessAccountType(msg.AccountType)
}
