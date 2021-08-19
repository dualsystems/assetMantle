/*
 Copyright [2019] - [2020], PERSISTENCE TECHNOLOGIES PTE. LTD. and the assetMantle contributors
 SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/CosmWasm/wasmd/app"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	serverTypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/snapshots"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/persistenceOne/persistenceSDK/schema/applications"
	"github.com/persistenceOne/persistenceSDK/schema/applications/base"
	"github.com/spf13/cast"
	"io"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	serverCmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/store"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	authClient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authCmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilCli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/persistenceOne/assetMantle/application"
	applicationCmd "github.com/persistenceOne/assetMantle/application/cmd"
	"github.com/spf13/cobra"
	tendermintCli "github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	tendermintDB "github.com/tendermint/tm-db"
)

func main() {
	configuration := sdkTypes.GetConfig()
	configuration.SetBech32PrefixForAccount(sdkTypes.Bech32PrefixAccAddr, sdkTypes.Bech32PrefixAccPub)
	configuration.SetBech32PrefixForValidator(sdkTypes.Bech32PrefixValAddr, sdkTypes.Bech32PrefixValPub)
	configuration.SetBech32PrefixForConsensusNode(sdkTypes.Bech32PrefixConsAddr, sdkTypes.Bech32PrefixConsPub)
	configuration.Seal()

	cobra.EnableCommandSorting = false

	encodingConfig := application.MakeEncodingConfig()
	initClientCtx := client.Context{}.
		WithJSONMarshaler(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.LegacyAmino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(application.DefaultNodeHome)

	rootCmd := &cobra.Command{
		Use:   application.Name,
		Short: "Asset Mantle App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			return server.InterceptConfigsPreRunHandler(cmd)
		},
	}

	authClient.Codec = encodingConfig.Marshaler

	cfg := sdkTypes.GetConfig()
	cfg.Seal()

	rootCmd.AddCommand(
		genutilCli.InitCmd(application.ModuleBasicManager, application.DefaultNodeHome),
		genutilCli.CollectGenTxsCmd(bankTypes.GenesisBalancesIterator{}, application.DefaultNodeHome),
		application.MigrateGenesisCmd(),
		genutilCli.GenTxCmd(application.ModuleBasicManager, encodingConfig.TxConfig, bankTypes.GenesisBalancesIterator{}, application.DefaultNodeHome),
		genutilCli.ValidateGenesisCmd(application.ModuleBasicManager),
		applicationCmd.AddGenesisAccountCommand(application.DefaultNodeHome),
		tendermintCli.NewCompletionCmd(rootCmd, true),
		// TODO testnetCmd(application.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
	)

	server.AddCommands(rootCmd, application.DefaultNodeHome, newApp, createSimAppAndExport, addModuleInitFlags)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		rpc.StatusCommand(),
		queryCommand(),
		txCommand(),
		keys.Commands(application.DefaultNodeHome),
	)

	if err := serverCmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}

func newApp(logger log.Logger, db tendermintDB.DB, traceStore io.Writer, appOpts serverTypes.AppOptions) serverTypes.Application {
	var cache sdkTypes.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get(server.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotDir := filepath.Join(cast.ToString(appOpts.Get(flags.FlagHome)), "data", "snapshots")
	snapshotDB, err := sdkTypes.NewLevelDB("metadata", snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	return application.Prototype.Initialize(
		logger, db, traceStore, true, skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		application.MakeEncodingConfig(), // Ideally, we would reuse the one created by NewRootCmd.
		appOpts,
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(server.FlagMinGasPrices))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(server.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(server.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(server.FlagMinRetainBlocks))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(server.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(server.FlagIndexEvents))),
		baseapp.SetSnapshotStore(snapshotStore),
		baseapp.SetSnapshotInterval(cast.ToUint64(appOpts.Get(server.FlagStateSyncSnapshotInterval))),
		baseapp.SetSnapshotKeepRecent(cast.ToUint32(appOpts.Get(server.FlagStateSyncSnapshotKeepRecent))),
	)
}

func createSimAppAndExport(
	logger log.Logger, db tendermintDB.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailAllowedAddrs []string,
	appOpts serverTypes.AppOptions) (serverTypes.ExportedApp, error) {

	encCfg := application.MakeEncodingConfig() // Ideally, we would reuse the one created by NewRootCmd.
	encCfg.Marshaler = codec.NewProtoCodec(encCfg.InterfaceRegistry)
	var assetMantleApp applications.Application
	if height != -1 {
		assetMantleApp = base.NewApplication(application.SimAppName, application.ModuleBasicManager, encCfg, application.EnabledWasmProposalTypeList, application.ModuleAccountPermissions).Initialize(logger, db, traceStore, false, map[int64]bool{}, "", cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)), encCfg, appOpts)

		if err := assetMantleApp.LoadHeight(height); err != nil {
			return serverTypes.ExportedApp{}, err
		}
	} else {
		assetMantleApp = base.NewApplication(application.SimAppName, application.ModuleBasicManager, encCfg, application.EnabledWasmProposalTypeList, application.ModuleAccountPermissions).Initialize(logger, db, traceStore, true, map[int64]bool{}, "", cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)), encCfg, appOpts)
	}

	return assetMantleApp.ExportApplicationStateAndValidators(forZeroHeight, jailAllowedAddrs)
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authCmd.GetSignCommand(),
		authCmd.GetSignBatchCommand(),
		authCmd.GetMultiSignCommand(),
		authCmd.GetMultiSignBatchCmd(),
		authCmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authCmd.GetBroadcastCommand(),
		authCmd.GetEncodeCommand(),
		authCmd.GetDecodeCommand(),
	)

	application.ModuleBasicManager.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authCmd.GetAccountCmd(),
		rpc.ValidatorCommand(),
		rpc.BlockCommand(),
		authCmd.QueryTxsByEventsCmd(),
		authCmd.QueryTxCmd(),
	)

	application.ModuleBasicManager.AddQueryCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}
