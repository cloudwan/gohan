package goext

type IUtil interface {
	NewUUID() string
	GetTransaction(context Context) (ITransaction, bool)
}
