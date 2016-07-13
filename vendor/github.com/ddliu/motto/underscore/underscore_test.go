// Copyright 2014 dong<ddliuhb@gmail.com>.
// Licensed under the MIT license.
//
// Underscore addon for Motto
package underscore

import (
	"github.com/ddliu/motto"
	"testing"
)

func TestUnderscoreImport(t *testing.T) {
	_, v, _ := motto.Run("tests/index.js")
	i, _ := v.ToInteger()

	if i != 1 {
		t.Error("import underscore test failed")
	}
}
