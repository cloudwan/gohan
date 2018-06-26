// Copyright 2014 dong<ddliuhb@gmail.com>.
// Licensed under the MIT license.
// 
// The Motto command line tool
package main

import (
    "fmt"
    "os"
    "github.com/ddliu/motto"
)

func usage() {
    fmt.Println("Usage: otto file.js")
    os.Exit(2)
}

func main() {
    if len(os.Args) != 2 {
        usage()
    }

    motto.Run(os.Args[1])
}