etcetera
========

This is an etcd client that uses a tagged struct to save and load values. The idea is to make it
easy to save and load our configuration structure in etcd.

To use it, simple add a 'etcd' tag to your structure mapping the attribute to an etcd URI. For now
you can add a tag in the following types:

  * struct
  * map[string]string
  * slice (of types struct, string, int, int64 and bool)
  * string
  * int
  * int64
  * bool
