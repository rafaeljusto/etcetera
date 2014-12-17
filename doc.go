// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package etcetera is etcd client that uses a tagged struct to save and load values
//
// How to use it
//
// Lets see an example to understand how it works. Imagine that your system today use a structure
// for configuration everything and it is persisted in a JSON file.
//
//   type B struct {
//     SubField1 string `json:"subfield1"`
//   }
//
//   type A struct {
//     Field1 string            `json:"field1"`
//     Field2 int               `json:"field2"`
//     Field3 int64             `json:"field3"`
//     Field4 bool              `json:"field4"`
//     Field5 B                 `json:"field5"`
//     Field6 map[string]string `json:"field6"`
//     Field7 []string          `json:"field7"`
//   }
//
// Now you want to start using etcd for configuration management. But the problem is that etcd works
// with URI and key/value, and you will need to change the way your configuration was developed to fit
// this style. Here is where this library will help you! It will map each field of the structure into
// an URI of etcd using tags as it is for JSON. Lets look our example:
//
//   type B struct {
//     SubField1 string `etcd:"/subfield1"`
//   }
//
//   type A struct {
//     Field1 string            `etcd:"/field1"`
//     Field2 int               `etcd:"/field2"`
//     Field3 int64             `etcd:"/field3"`
//     Field4 bool              `etcd:"/field4"`
//     Field5 B                 `etcd:"/field5"`
//     Field6 map[string]string `etcd:"/field6"`
//     Field7 []string          `etcd:"/field7"`
//   }
//
// And that's it! You can still work with your structure and now have the flexibility of a centralized
// configuration system. The best part is that you can also monitor some field for changes, calling a
// callback when something happens.
//
// For now you can add a tag in the following types:
//
//   * struct
//   * map[string]string
//   * map[string]struct
//   * []string
//   * []struct
//   * []int
//   * []int64
//   * []bool
//   * string
//   * int
//   * int64
//   * bool
//
// When saving or loading a structure, attributes without the tag 'etcd' or other types from the listed
// above are going to be ignored.
//
// Examples
//
//   type B struct {
//     SubField1 string `etcd:"/subfield1"`
//   }
//
//   type A struct {
//     Field1 string            `etcd:"/field1"`
//     Field2 int               `etcd:"/field2"`
//     Field3 int64             `etcd:"/field3"`
//     Field4 bool              `etcd:"/field4"`
//     Field5 B                 `etcd:"/field5"`
//     Field6 map[string]string `etcd:"/field6"`
//     Field7 []string          `etcd:"/field7"`
//   }
//
//   func ExampleSave() {
//     a := A{
//       Field1: "value1",
//       Field2: 10,
//       Field3: 999,
//       Field4: true,
//       Field5: B{"value2"},
//       Field6: map[string]string{"key1": "value3"},
//       Field7: []string{"value4", "value5", "value6"},
//     }
//
//     client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
//     if err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     if err := client.Save(); err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     fmt.Printf("%+v\n", a)
//   }
//
//   func ExampleLoad() {
//     var a A
//
//     client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
//     if err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     if err := client.Load(); err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     fmt.Printf("%+v\n", a)
//   }
//
//   func ExampleWatch() {
//     var a A
//
//     client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
//     if err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     stop, err := client.Watch(a.Field1, func() {
//       fmt.Printf("%+v\n", a)
//     })
//
//     if err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     close(stop)
//   }
//
//   func ExampleVersion() {
//     var a A
//
//     client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
//     if err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     if err := client.Load(); err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     version, err := client.Version(&a.Field1)
//     if err != nil {
//       fmt.Println(err.Error())
//       return
//     }
//
//     fmt.Printf("%d\n", version)
//   }
package etcetera
