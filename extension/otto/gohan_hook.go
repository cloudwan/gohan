package otto

import (
	"github.com/xyproto/otto"
)

func init() {
	gohanHookInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_load_hook": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_load_hook", 1)
				funcName, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				env.loadHooks = append(env.loadHooks, funcName)
				return otto.TrueValue()
			},
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}

	}
	RegisterInit(gohanHookInit)
}
