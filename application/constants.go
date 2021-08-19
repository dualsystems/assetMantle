/*
 Copyright [2019] - [2020], PERSISTENCE TECHNOLOGIES PTE. LTD. and the assetMantle contributors
 SPDX-License-Identifier: Apache-2.0
*/

package application

import (
	"os"
)

const Name = "AssetMantle"
const SimAppName = "SimAssetMantle"

var DefaultNodeHome = os.ExpandEnv("$HOME/." + Name)
