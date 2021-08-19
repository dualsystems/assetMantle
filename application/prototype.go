/*
 Copyright [2019] - [2020], PERSISTENCE TECHNOLOGIES PTE. LTD. and the assetMantle contributors
 SPDX-License-Identifier: Apache-2.0
*/

package application

import (
	"github.com/persistenceOne/persistenceSDK/schema/applications/base"
)

var Prototype = base.NewApplication(
	Name,
	ModuleBasicManager,
	MakeEncodingConfig(),
	EnabledWasmProposalTypeList,
	ModuleAccountPermissions,
)
