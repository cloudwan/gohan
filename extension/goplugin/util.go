// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goplugin

import (
	"fmt"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/golang/mock/gomock"
	"github.com/twinj/uuid"
)

type Util struct {
}

func contextGetTransaction(ctx goext.Context) (goext.ITransaction, bool) {
	ctxTx := ctx["transaction"]
	if ctxTx == nil {
		return nil, false
	}

	switch tx := ctxTx.(type) {
	case goext.ITransaction:
		return tx, true
	case transaction.Transaction:
		return &Transaction{tx}, true
	default:
		panic(fmt.Sprintf("Unknown transaction type in context: %+v", ctxTx))
	}
}

// NewUUID create a new unique ID
func (util *Util) NewUUID() string {
	return uuid.NewV4().String()
}

func (u *Util) GetTransaction(context goext.Context) (goext.ITransaction, bool) {
	return contextGetTransaction(context)
}

func (u *Util) Clone() *Util {
	return &Util{}
}

var controllers map[gomock.TestReporter]*gomock.Controller = make(map[gomock.TestReporter]*gomock.Controller)

func NewController(testReporter gomock.TestReporter) *gomock.Controller {
	ctrl := gomock.NewController(testReporter)
	controllers[testReporter] = ctrl
	return ctrl
}

func Finish(testReporter gomock.TestReporter) {
	controllers[testReporter].Finish()
}
