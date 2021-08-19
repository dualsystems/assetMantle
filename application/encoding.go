package application

import (
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/persistenceOne/persistenceSDK/schema/applications/base/encoding"
)

func MakeEncodingConfig() encoding.EncodingConfig {
	encodingConfig := encoding.MakeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.LegacyAmino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasicManager.RegisterLegacyAminoCodec(encodingConfig.LegacyAmino)
	ModuleBasicManager.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
