// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"fmt"

	"github.com/coreos/go-etcd/etcd"
)

func ExampleSaveLoad() {
	type A struct {
		Field1 string            `etcd:"/field1"`
		Field2 int               `etcd:"/field2"`
		Field3 int64             `etcd:"/field3"`
		Field4 bool              `etcd:"/field4"`
		Field5 B                 `etcd:"/field5"`
		Field6 map[string]string `etcd:"/field6"`
		Field7 []string          `etcd:"/field7"`
	}

	type B struct {
		SubField1 string `etcd:"/subfield1"`
	}

	a1 := A{
		Field1: "value1",
		Field2: 10,
		Field3: 999,
		Field4: true,
		Field5: B{
			SubField1: "value2",
		},
		Field6: map[string]string{
			"key1": "value3",
		},
		Field7: []string{"value4", "value5", "value6"},
	}

	client := etcd.NewClient([]string{"http://127.0.0.1:4001"})

	if err := Save(&a1, client); err != nil {
		fmt.Println(err.Error())
	}

	a2 := A{
		Field6: make(map[string]string),
	}

	if err := Load(&a2, client); err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("Input: %+v\n", a1)
	fmt.Printf("Output: %+v\n", a2)
}
