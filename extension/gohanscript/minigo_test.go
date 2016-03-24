package gohanscript_test

import (
	"fmt"

	"github.com/cloudwan/gohan/extension/gohanscript"
	//Import gohan script lib
	_ "github.com/cloudwan/gohan/extension/gohanscript/autogen"

	. "github.com/onsi/ginkgo"
)

func testExpr(context *gohanscript.Context, expr string, expected interface{}) {
	vm := gohanscript.NewVM()
	minigo, err := gohanscript.CompileExpr(expr)
	if err != nil {
		Fail(fmt.Sprintf("Parse failed expr: %s failed with %s", expr, err))
		return
	}

	actual, err := minigo.Run(vm, context)
	if err != nil {
		Fail(fmt.Sprintf("Eval failed expr: %s failed with %s", expr, err))
		return
	}
	if f, ok := expected.(float64); ok {
		if f-actual.(float64) > 0.0001 {
			Fail(fmt.Sprintf("Expr: %s, expected: %v, actual: %v", expr, expected, actual))
			return
		}
		return
	}
	if expected != actual {
		Fail(fmt.Sprintf("Expr: %s, expected: %v, actual: %v", expr, expected, actual))
		return
	}
}

func testGoStmt(context *gohanscript.Context, stmt string, expected interface{}) {
	vm := gohanscript.NewVM()
	minigo, err := gohanscript.CompileGoStmt(stmt)
	if err != nil {
		Fail(fmt.Sprintf("Parse failed stmt: %s failed with %s", stmt, err))
		return
	}
	actual, err := minigo.Run(vm, context)
	if err != nil {
		Fail(fmt.Sprintf("Eval failed stmt: %s failed with %s", stmt, err))
		return
	}
	if expected != actual {
		Fail(fmt.Sprintf("Stmt: %s, expected: %v, actual: %v", stmt, expected, actual))
		return
	}
}

var _ = Describe("Run minigo test", func() {
	Context("When given expresstion", func() {
		It("All test should be passed", func() {
			context := gohanscript.NewContext()
			context.Set("a", 1)
			context.Set("b", 3.14)
			testExpr(context, "a+1", 2)
			testExpr(context, "a-1", 0)
			testExpr(context, "2*3", 6)
			testExpr(context, "2*3 - 1", 5)
			testExpr(context, "(3 - 1) * ( 2 - 1 ) ", 2)
			testExpr(context, "10 / 5", 2)
			testExpr(context, "b+1.0", 4.14)
			testExpr(context, "b-1.0", 2.14)
			testExpr(context, "2.3*3.0", 6.9)
			testExpr(context, "2.1 - 1.1", 1.0)
			testExpr(context, "2.1 / 3.0", 0.7)
			testExpr(context, "5 % 2", 1)
			context.Set("list", []interface{}{1, 2, 3})
			testExpr(context, "len(list) == 3", true)
			testExpr(context, "2 > 1", true)
			testExpr(context, "1 > 1", false)
			testExpr(context, "1 < 2", true)
			testExpr(context, "1 < 1", false)
			testExpr(context, "2 >= 1", true)
			testExpr(context, "1 >= 1", true)
			testExpr(context, "0 >= 1", false)
			testExpr(context, "1 <= 2", true)
			testExpr(context, "1 <= 1", true)
			testExpr(context, "1 <= 0", false)
			testExpr(context, "1 == 1", true)
			testExpr(context, "1 != 1", false)
			testExpr(context, "1000 != -1", true)
			testExpr(context, "-1 == -1", true)
			testExpr(context, "true && true", true)
			testExpr(context, "false && true", false)
			testExpr(context, "true && false", false)
			testExpr(context, "false && false", false)
			testExpr(context, "true || true", true)
			testExpr(context, "false || true", true)
			testExpr(context, "true || false", true)
			testExpr(context, "false || false", false)
			testExpr(context, "!true", false)
			testExpr(context, "!false", true)
			testExpr(context, "1 & 0", 0)
			testExpr(context, "1 | 0", 1)
			testExpr(context, "1 ^ 0", 1)
			context.Set("b", "app")
			testExpr(context, `b + "le"`, "apple")
		})
	})
	Context("When given statement", func() {
		It("All test should be passed", func() {
			context := gohanscript.NewContext()
			context.Set("list", []interface{}{10, 20, 30})
			context.Set("m", map[string]interface{}{"a": 1, "b": 2})
			testGoStmt(context, `
                a := 1
                b := 2
                return a + b`,
				3)
			testGoStmt(context, `
                a := 2
                b := 1
                if a > b {
                    return true
                }
                return false`,
				true)
			testGoStmt(context, `
                a := 0
                for i := 0; i <= 10; i++ {
                    a = a + i
                }
                return a`,
				55)
			testGoStmt(context, `
                a := 0
                for i := 10; i > 0; i-- {
                    a = a + i
                }
                return a`,
				55)
			testGoStmt(context, `
                list[2] = list[0] + list[1] + list[2]
                return list[2]`,
				60)
			testGoStmt(context, `
                m["c"] = m["a"] + m["b"]
                return m["c"]`,
				3)

			testGoStmt(context, `
                i := 0
                for _, value := range m {
                    i = i + value
                }
                return i`,
				6)
			context.Set("list", []interface{}{10, 20, 30})
			testGoStmt(context, `
                i := 0
                for _, value := range list {
                    i = i + value
                }
                return i`,
				60)
			testGoStmt(context, `
                i := 0
                for index := range list {
                    i = i + index
                }
                return i`,
				3)

			testGoStmt(context, `
                a := 1
                b := 2
                if a > b {
                    return false
                }
                return true`,
				true)
			testGoStmt(context, `
                a := 1
                b := 2
                if a < b {
                    return true
                }
                return false`,
				true)
			testGoStmt(context, `
                a := 1
                b := 2
                if a > 3 {
                    return false
                } else if a < b {
                    return true
                }
                return false`,
				true)

		})
	})
})
