etcetera
========

[![Build Status](https://travis-ci.org/rafaeljusto/etcetera.png?branch=master)](https://travis-ci.org/rafaeljusto/etcetera)
[![Coverage Status](https://img.shields.io/coveralls/rafaeljusto/etcetera.svg)](https://coveralls.io/r/rafaeljusto/etcetera)
[![GoDoc](https://godoc.org/github.com/rafaeljusto/etcetera?status.png)](https://godoc.org/github.com/rafaeljusto/etcetera)

This is an [etcd](https://coreos.com/using-coreos/etcd/) client that uses a tagged struct to save
and load values. The library is only an extra layer of abstraction over the
[go-etcd](http://github.com/coreos/go-etcd) library. The idea was originally from Gustavo Henrique
Montesião de Sousa ([@gustavo-hms](https://github.com/gustavo-hms)).

How to use it
-------------

```
go get -u github.com/rafaeljusto/etcetera
```

This project has the following dependencies:
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
