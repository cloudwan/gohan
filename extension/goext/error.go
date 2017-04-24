package goext

type ErrNo int

type Error struct {
	Code ErrNo
	Err  error
}

func (e Error) Error() string {
	return string(e.Code) + e.Err.Error()
}

var (
	ErrBadRequest          = ErrNo(400)
	ErrConflict            = ErrNo(409)
	ErrNotFound            = ErrNo(404)
	ErrInternalServerError = ErrNo(500)
	ErrNotImplemented      = ErrNo(501)
)
