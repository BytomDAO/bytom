# [strftime](http://strftime.org/) for Go

[![Test](https://github.com/tebeka/strftime/workflows/Test/badge.svg)](https://github.com/tebeka/strftime/actions?query=workflow%3ATest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/tebeka/strftime)


Q: Why? We already have [time.Format](https://golang.org/pkg/time/#Time.Format).

A: Yes, but it becomes tricky to use if if you have string with things other
than time in them. (like `/path/to/%Y/%m/%d/report`)


# Installing

    go get github.com/tebeka/strftime

# Example

    str, err := strftime.Format("%Y/%m/%d", time.Now())

# Contact
https://github.com/tebeka/strftime
