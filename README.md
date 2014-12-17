etcetera
========

[![Build Status](https://travis-ci.org/rafaeljusto/etcetera.png?branch=master)](https://travis-ci.org/rafaeljusto/etcetera)
[![Coverage Status](https://img.shields.io/coveralls/rafaeljusto/etcetera.svg)](https://coveralls.io/r/rafaeljusto/etcetera)
[![GoDoc](https://godoc.org/github.com/rafaeljusto/etcetera?status.png)](https://godoc.org/github.com/rafaeljusto/etcetera)

This is an [etcd](https://coreos.com/using-coreos/etcd/) client that uses a tagged struct to save
and load values from the etcd cluster. etcetera is only an abstraction layer over the
[go-etcd](http://github.com/coreos/go-etcd) library. It was designed to be simple to use and make
transitions from JSON to key-value configuration easier.

The idea was originally from my co-worker Gustavo Henrique Montesião de Sousa
([@gustavo-hms](https://github.com/gustavo-hms)).

How to use it
-------------

Download the library using the command bellow.

```
go get -u github.com/rafaeljusto/etcetera
```

This project has the following dependencies:
  * github.com/coreos/go-etcd/etcd

So you should also download the dependencies.

```
go get -u github.com/coreos/go-etcd/etcd
```

Now lets see an example to understand how it works. Imagine that your system today use a structure
for configuration everything and it is persisted in a JSON file.

```go
type B struct {
  SubField1 string `json:"subfield1"`
}

type A struct {
  Field1 string            `json:"field1"`
  Field2 int               `json:"field2"`
  Field3 int64             `json:"field3"`
  Field4 bool              `json:"field4"`
  Field5 B                 `json:"field5"`
  Field6 map[string]string `json:"field6"`
  Field7 []string          `json:"field7"`
}
```

Now you want to start using etcd for configuration management. But the problem is that etcd works
with URI and key/value, and you will need to change the way your configuration was developed to fit
this style. Here is where this library will help you! It will map each field of the structure into
an URI of etcd using tags as it is for JSON. Lets look our example:

```go
type B struct {
  SubField1 string `etcd:"subfield1"`
}

type A struct {
  Field1 string            `etcd:"field1"`
  Field2 int               `etcd:"field2"`
  Field3 int64             `etcd:"field3"`
  Field4 bool              `etcd:"field4"`
  Field5 B                 `etcd:"field5"`
  Field6 map[string]string `etcd:"field6"`
  Field7 []string          `etcd:"field7"`
}
```

And that's it! You can still work with your structure and now have the flexibility of a centralized
configuration system. The best part is that you can also monitor some field for changes, calling a
callback when something happens.

What happens is that the library will build the URI of etcd based on the tags, so if we want to look
for the "A.FieldB.SubField1" field in etcd we would have to look at the URI "/field5/subfield1".
Now, using just this strategy would limit to have only one configuration structure in the etcd
cluster, for that reason you can define a namespace in the constructor. For example, when checking
the same field "A.FieldB.SubField1" with the namespace "test" the URI to look for would be
"/test/field5/subfield1".

For now you can add a tag in the following types:

  * struct
  * map[string]string
  * map[string]struct
  * []string
  * []struct
  * []int
  * []int64
  * []bool
  * string
  * int
  * int64
  * bool

When saving or loading a structure, attributes without the tag 'etcd' or other types from the listed
above are going to be ignored.

Performance
-----------

To make the magic we use reflection, and this can degrade performance. But the purpouse is to use
this library to centralize the configurations of your project into a etcd cluster, and for this the
performance isn't the most important issue. Here are some benchmarks (without etcd I/O and latency
delays):

```
BenchmarkSave  2000000         710 ns/op
BenchmarkLoad  2000000         625 ns/op
BenchmarkWatch  300000        4890 ns/op
```

Fill free to send pull requests to improve the performance or make the code cleaner (I will thank
you a lot!). Just remember to run the tests after every code change.

Examples
--------

```go
type B struct {
  SubField1 string `etcd:"subfield1"`
}

type A struct {
  Field1 string            `etcd:"field1"`
  Field2 int               `etcd:"field2"`
  Field3 int64             `etcd:"field3"`
  Field4 bool              `etcd:"field4"`
  Field5 B                 `etcd:"field5"`
  Field6 map[string]string `etcd:"field6"`
  Field7 []string          `etcd:"field7"`
}

func ExampleSave() {
  a := A{
    Field1: "value1",
    Field2: 10,
    Field3: 999,
    Field4: true,
    Field5: B{"value2"},
    Field6: map[string]string{"key1": "value3"},
    Field7: []string{"value4", "value5", "value6"},
  }

  client, err := NewClient([]string{"http://127.0.0.1:4001"}, "test", &a)
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  if err := client.Save(); err != nil {
    fmt.Println(err.Error())
    return
  }

  fmt.Printf("%+v\n", a)
}

func ExampleLoad() {
  var a A

  client, err := NewClient([]string{"http://127.0.0.1:4001"}, "test", &a)
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  if err := client.Load(); err != nil {
    fmt.Println(err.Error())
    return
  }

  fmt.Printf("%+v\n", a)
}

func ExampleWatch() {
  var a A

  client, err := NewClient([]string{"http://127.0.0.1:4001"}, "test", &a)
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  stop, err := client.Watch(a.Field1, func() {
    fmt.Printf("%+v\n", a)
  })

  if err != nil {
    fmt.Println(err.Error())
    return
  }

  close(stop)
}

func ExampleVersion() {
  var a A

  client, err := NewClient([]string{"http://127.0.0.1:4001"}, "test", &a)
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  if err := client.Load(); err != nil {
    fmt.Println(err.Error())
    return
  }

  version, err := client.Version(&a.Field1)
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  fmt.Printf("%d\n", version)
}
```