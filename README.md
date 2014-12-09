etcetera
========

[![Build Status](https://travis-ci.org/rafaeljusto/etcetera.png?branch=master)](https://travis-ci.org/rafaeljusto/etcetera)

This is an etcd client that uses a tagged struct to save and load values. The idea was originally
from Gustavo Henrique Montesião de Sousa (@gustavo-hms).

To use it, simple add a 'etcd' tag to your structure mapping the attribute to an etcd URI. For now
you can add a tag in the following types:

  * struct
  * map[string]string
  * slice (of types struct, string, int, int64 and bool)
  * string
  * int
  * int64
  * bool
