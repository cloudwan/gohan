// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package golang

import (
	"github.com/cloudwan/gohan/db/transaction"
//	"github.com/cloudwan/gohan/extension/golang"
)

func DbTransaction(env *Environment) transaction.Transaction {
	//tx, err := rawEnvironment.Env.(golang.Environment).DataStore.Begin()
	//if err != nil {
		//ThrowOttoException(&call, "failed to start a transaction: %s", err.Error())
	//}
	//return tx
	return nil
}
