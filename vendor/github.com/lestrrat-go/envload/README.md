# envload

Restore environment variables, so you can break 'em

[![Build Status](https://travis-ci.org/lestrrat-go/envload.png?branch=master)](https://travis-ci.org/lestrrat-go/envload)

[![GoDoc](https://godoc.org/github.com/lestrrat-go/envload?status.svg)](https://godoc.org/github.com/lestrrat-go/envload)

# SYNOPSIS

# DESCRIPTION

Certain applications that require reloading of configuraiton from
environment variables are sensitive to these values being changed.

Or maybe you are writing a test that wants to temporarily change the
value of an environment variable, but you don't want it to linger afterwards.

In other languages this can be done with a "temporary" variable, like in
Perl5:

```perl
use strict;
use 5.24;

sub foo {
  $ENV{IMPORTANT_VAR} = "foo";
  say $ENV{IMPORTANT_VAR}; # "foo"

  {
    local %ENV = %ENV; # inherit the original %ENV,
    $ENV{IMPORTANT_VAR} = "bar";
    say $ENV{IMPORTANT_VAR}; # "bar"
  }
  # 

  say $ENV{IMPORTANT_VAR}; # "bar"
}
```

This library basically allows you to do this in Go
