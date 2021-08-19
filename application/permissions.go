package application

import (
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distributionTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govTypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ibcTransferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	mintTypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/persistenceOne/persistenceSDK/modules/splits"
)

var ModuleAccountPermissions = map[string][]string{
	authTypes.FeeCollectorName:     nil,
	distributionTypes.ModuleName:   nil,
	mintTypes.ModuleName:           {authTypes.Minter},
	stakingTypes.BondedPoolName:    {authTypes.Burner, authTypes.Staking},
	stakingTypes.NotBondedPoolName: {authTypes.Burner, authTypes.Staking},
	govTypes.ModuleName:            {authTypes.Burner},
	ibcTransferTypes.ModuleName:    {authTypes.Minter, authTypes.Burner},
	splits.Prototype().Name():      nil,
}
