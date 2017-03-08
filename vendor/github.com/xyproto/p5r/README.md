<img src="https://raw.github.com/xyproto/p5r/master/logo.png" style="margin-left: 2em">

# P5R

This is a fork of [regexp2](https://github.com/dlclark/regexp2).

The motivation is to make Perl 5 compatible regular expressions available to [Otto](https://github.com/robertkrimen/otto), an implementation of JavaScript for Go.

Like `regexp2`, this package does not have constant time guarantees like the `regexp` package in the standard library, but it allows backtracking and is compatible with Perl 5 regular expressions.

`regexp2` was inspired by the regular expression implementation in .NET (which is released under an MIT license).

The main difference from regexp2 is the renaming and modification of function signatures to be more compatible with how [Otto](https://github.com/robertkrimen/otto) is using regular expressions. 

Example usage:

```go
re := p5r.MustCompile(`Your pattern`)
if isMatch := re.MatchString(`Something to match`); isMatch {
    //do something
}
```

For more information, take a look at the [regexp2](https://github.com/dlclark/regexp2) README.md.


---

License: MIT
