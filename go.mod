module github.com/persistenceOne/assetMantle

go 1.14

require (
	github.com/CosmWasm/wasmd v0.17.0
	github.com/cosmos/cosmos-sdk v0.42.6
	github.com/persistenceOne/persistenceSDK v0.1.1-0.20210628145202-7c8c7ff1e223
	github.com/pkg/errors v0.9.1
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/tendermint/tendermint v0.34.11
	github.com/tendermint/tm-db v0.6.4
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/persistenceOne/persistenceSDK => ../persistenceSDK
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
