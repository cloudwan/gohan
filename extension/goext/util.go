package goext

type IUtil interface {
	GetTransaction(context Context) (ITransaction, bool)
}
