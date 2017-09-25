package goplugin

import (
	"fmt"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
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
