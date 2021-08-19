/*
 Copyright [2019] - [2020], PERSISTENCE TECHNOLOGIES PTE. LTD. and the assetMantle contributors
 SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingTypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilTypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagClientHome   = "home-client"
	flagVestingStart = "vesting-start-time"
	flagVestingEnd   = "vesting-end-time"
	flagVestingAmt   = "vesting-amount"
)

// AddGenesisAccountCommand returns add-genesis-account cobra Command.
func AddGenesisAccountCommand(defaultNodeHome string) *cobra.Command {
	command := &cobra.Command{
		Use:   "add-genesis-account [address_or_key_name] [coin][,[coin]]",
		Short: "Add a genesis account to genesis.json",
		Long: `Add a genesis account to genesis.json. The provided account must specify
the account address or key name and a list of initial coins. If a key name is given,
the address will be looked up in the local Keybase. The list of initial tokens must
contain valid denominations. Accounts may optionally be supplied with vesting parameters.
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			depCdc := clientCtx.JSONMarshaler
			codec := depCdc.(codec.Marshaler)

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			config.SetRoot(clientCtx.HomeDir)

			addr, err := sdkTypes.AccAddressFromBech32(args[0])
			inBuf := bufio.NewReader(cmd.InOrStdin())
			if err != nil {
				// attempt to lookup address from Keybase if no address was provided
				kb, err := keyring.New(
					sdkTypes.KeyringServiceName(),
					viper.GetString(flags.FlagKeyringBackend),
					viper.GetString(flagClientHome),
					inBuf,
				)
				if err != nil {
					return err
				}

				info, err := kb.Key(args[0])
				if err != nil {
					return fmt.Errorf("failed to get address from Keybase: %w", err)
				}

				addr = info.GetAddress()
			}

			coins, err := sdkTypes.ParseCoinsNormalized(args[1])
			if err != nil {
				return fmt.Errorf("failed to parse coins: %w", err)
			}

			vestingStart, err := cmd.Flags().GetInt64(flagVestingStart)
			if err != nil {
				return err
			}
			vestingEnd, err := cmd.Flags().GetInt64(flagVestingEnd)
			if err != nil {
				return err
			}
			vestingAmtStr, err := cmd.Flags().GetString(flagVestingAmt)
			if err != nil {
				return err
			}

			vestingAmt, err := sdkTypes.ParseCoinsNormalized(vestingAmtStr)
			if err != nil {
				return fmt.Errorf("failed to parse vesting amount: %w", err)
			}

			// create concrete account type based on input parameters
			var genAccount authTypes.GenesisAccount

			balances := bankTypes.Balance{Address: addr.String(), Coins: coins.Sort()}
			baseAccount := authTypes.NewBaseAccount(addr, nil, 0, 0)

			if !vestingAmt.IsZero() {
				baseVestingAccount := vestingTypes.NewBaseVestingAccount(baseAccount, vestingAmt.Sort(), vestingEnd)
				if err != nil {
					return fmt.Errorf("failed to create base vesting account: %w", err)
				}

				if (balances.Coins.IsZero() && !baseVestingAccount.OriginalVesting.IsZero()) ||
					baseVestingAccount.OriginalVesting.IsAnyGT(balances.Coins) {
					return errors.New("vesting amount cannot be greater than total amount")
				}

				switch {
				case vestingStart != 0 && vestingEnd != 0:
					genAccount = vestingTypes.NewContinuousVestingAccountRaw(baseVestingAccount, vestingStart)

				case vestingEnd != 0:
					genAccount = vestingTypes.NewDelayedVestingAccountRaw(baseVestingAccount)

				default:
					return errors.New("invalid vesting parameters; must supply start and end time or end time")
				}
			} else {
				genAccount = baseAccount
			}

			if err := genAccount.Validate(); err != nil {
				return fmt.Errorf("failed to validate new genesis account: %w", err)
			}

			genFile := config.GenesisFile()
			appState, genDoc, err := genutilTypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			authGenState := authTypes.GetGenesisStateFromAppState(codec, appState)

			accounts, err := authTypes.UnpackAccounts(authGenState.Accounts)
			if err != nil {
				return fmt.Errorf("failed to get accounts from any: %w", err)
			}

			if accounts.Contains(addr) {
				return fmt.Errorf("cannot add account at existing address %s", addr)
			}

			// Add the new account to the set of genesis accounts and sanitize the
			// accounts afterwards.
			accounts = append(accounts, genAccount)
			accounts = authTypes.SanitizeGenesisAccounts(accounts)

			genAccs, err := authTypes.PackAccounts(accounts)
			if err != nil {
				return fmt.Errorf("failed to convert accounts into any's: %w", err)
			}

			authGenState.Accounts = genAccs

			authGenStateBz, err := codec.MarshalJSON(&authGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal auth genesis state: %w", err)
			}

			appState[authTypes.ModuleName] = authGenStateBz

			bankGenState := bankTypes.GetGenesisStateFromAppState(depCdc, appState)
			bankGenState.Balances = append(bankGenState.Balances, balances)
			bankGenState.Balances = bankTypes.SanitizeGenesisBalances(bankGenState.Balances)
			bankGenState.Supply = bankGenState.Supply.Add(balances.Coins...)

			bankGenStateBz, err := codec.MarshalJSON(bankGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal bank genesis state: %w", err)
			}

			appState[bankTypes.ModuleName] = bankGenStateBz

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	command.Flags().String(flags.FlagHome, defaultNodeHome, "The application's home directory")
	command.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring backend (os|file|test)")
	command.Flags().String(flagVestingAmt, "", "amount of coins for vesting accounts")
	command.Flags().Uint64(flagVestingStart, 0, "schedule start time (unix epoch) for vesting accounts")
	command.Flags().Uint64(flagVestingEnd, 0, "schedule end time (unix epoch) for vesting accounts")

	return command
}
