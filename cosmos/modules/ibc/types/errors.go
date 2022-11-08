package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// IBC transfer sentinel errors
var (

	// Due to the conflicting types in osmosis cosmos ibc-go (v2) and ibc-go v3, if you call sdkerrors.Register (as would normally be done)
	// it will PANIC. Since we just want to be allowed to unmarshal, it doesn't matter for purposes of our client.
	// In other words... we are forcing a duplicate type registration that would normally cause a panic. See sdkerrors.Register code.
	ErrInvalidDenomForTransfer = sdkerrors.New(ModuleName, 3, "invalid denomination for cross-chain transfer")

	// ErrInvalidPacketTimeout    = sdkerrors.Register(ModuleName, 2, "invalid packet timeout")
	// ErrInvalidVersion          = sdkerrors.Register(ModuleName, 4, "invalid ICS20 version")
	// ErrInvalidAmount           = sdkerrors.Register(ModuleName, 5, "invalid token amount")
	// ErrTraceNotFound           = sdkerrors.Register(ModuleName, 6, "denomination trace not found")
	// ErrSendDisabled            = sdkerrors.Register(ModuleName, 7, "fungible token transfers from this chain are disabled")
	// ErrReceiveDisabled         = sdkerrors.Register(ModuleName, 8, "fungible token transfers to this chain are disabled")
	// ErrMaxTransferChannels     = sdkerrors.Register(ModuleName, 9, "max transfer channels")
)
