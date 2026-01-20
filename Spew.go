package main

import (
	"github.com/davecgh/go-spew/spew"
)

// カスタム設定でインデント付き出力
var spewConfig = spew.ConfigState{
	Indent:                  "    ", // 4スペースインデント
	DisableMethods:          true,
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true, // マップのキーをソート
	SpewKeys:                true,
}
