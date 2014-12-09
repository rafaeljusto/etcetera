etcetera
========

[![Build Status](https://travis-ci.org/rafaeljusto/etcetera.png?branch=master)](https://travis-ci.org/rafaeljusto/etcetera)
[![GoDoc](https://godoc.org/github.com/rafaeljusto/etcetera?status.png)](https://godoc.org/github.com/rafaeljusto/etcetera)

This is an etcd client that uses a tagged struct to save and load values. The idea was originally
from Gustavo Henrique Montesião de Sousa (@gustavo-hms).

How to use it
-------------

```
go get -u github.com/rafaeljusto/etcetera
```

This project has the following dependencies:
  * github.com/coreos/etcd
  * github.com/coreos/go-etcd/etcd

To use it in your code, simple add a 'etcd' tag to your structure mapping the attribute to an etcd
URI. For now you can add a tag in the following types:

  * struct
  * map[string]string
  * slice (of types struct, string, int, int64 and bool)
  * string
  * int
  * int64
  * bool

When saving or loading a structure, attributes without the tag 'etcd' or other types from the listed
above are going to be ignored.
