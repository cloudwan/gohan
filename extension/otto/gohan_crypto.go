package otto

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash"
	"io"

	"github.com/ddliu/motto"
	"github.com/robertkrimen/otto"
)

func cryptoModule(vm *motto.Motto) (otto.Value, error) {
	mod, _ := vm.Object(`({})`)

	mod.Set("createHash", func(call otto.FunctionCall) otto.Value {
		VerifyCallArguments(&call, "createHash", 1)
		_, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Algorithm: %v", err)

		return wrapHash(md5.New(), vm)
	})

	return vm.ToValue(mod)
}

func wrapHash(h hash.Hash, vm *motto.Motto) otto.Value {
	wrapped, _ := vm.Object(`({})`)

	wrapped.Set("update", func(call otto.FunctionCall) otto.Value {
		data, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Data: %v", err)

		_, err = io.WriteString(h, data)
		ThrowWithMessageIfHappened(&call, err,
			"Failed to update hash with data: %v", err)

		v, _ := vm.ToValue(wrapped)
		return v
	})

	wrapped.Set("digest", func(call otto.FunctionCall) otto.Value {
		encoding, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Encoding: %v", err)

		sum := h.Sum(nil)

		var result string
		switch encoding {
		case "base64":
			result = base64.StdEncoding.EncodeToString(sum)
		case "hex":
			result = fmt.Sprintf("%x", sum)
		default:
			ThrowOttoException(&call,
				"Unsupported hash encoding: '%s'", encoding)
		}

		v, _ := vm.ToValue(result)
		return v
	})

	v, _ := vm.ToValue(wrapped)
	return v
}
