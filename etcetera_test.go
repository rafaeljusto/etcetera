// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/go-etcd/etcd"
)

const DEBUG = false

func ExampleSave() {
	type B struct {
		SubField1 string `etcd:"/subfield1"`
	}

	type A struct {
		Field1 string            `etcd:"/field1"`
		Field2 int               `etcd:"/field2"`
		Field3 int64             `etcd:"/field3"`
		Field4 bool              `etcd:"/field4"`
		Field5 B                 `etcd:"/field5"`
		Field6 map[string]string `etcd:"/field6"`
		Field7 []string          `etcd:"/field7"`
	}

	a := A{
		Field1: "value1",
		Field2: 10,
		Field3: 999,
		Field4: true,
		Field5: B{"value2"},
		Field6: map[string]string{"key1": "value3"},
		Field7: []string{"value4", "value5", "value6"},
	}

	client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
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
	type B struct {
		SubField1 string `etcd:"/subfield1"`
	}

	type A struct {
		Field1 string            `etcd:"/field1"`
		Field2 int               `etcd:"/field2"`
		Field3 int64             `etcd:"/field3"`
		Field4 bool              `etcd:"/field4"`
		Field5 B                 `etcd:"/field5"`
		Field6 map[string]string `etcd:"/field6"`
		Field7 []string          `etcd:"/field7"`
	}

	a := A{
		Field6: make(map[string]string),
	}

	client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
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
	type B struct {
		SubField1 string `etcd:"/subfield1"`
	}

	type A struct {
		Field1 string            `etcd:"/field1"`
		Field2 int               `etcd:"/field2"`
		Field3 int64             `etcd:"/field3"`
		Field4 bool              `etcd:"/field4"`
		Field5 B                 `etcd:"/field5"`
		Field6 map[string]string `etcd:"/field6"`
		Field7 []string          `etcd:"/field7"`
	}

	a := A{
		Field6: make(map[string]string),
	}

	client, err := NewClient([]string{"http://127.0.0.1:4001"}, &a)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	_, err = client.Watch(a.Field1, func() {
		fmt.Printf("%+v\n", a)
	})

	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func TestNewClient(t *testing.T) {
	test := struct {
		Field1 string
		Field2 int `etcd:"/field2"`
	}{}

	data := []struct {
		description string      // describe the test case
		machines    []string    // etcd servers
		config      interface{} // configuration instance (structure) to save
		expectedErr bool        // error expectation when building the object
		expected    Client      // expected client object after calling the constructor
	}{
		{
			description: "it should create a valid Client object",
			machines: []string{
				"http://127.0.0.1:4001",
				"http://127.0.0.1:4002",
				"http://127.0.0.1:4003",
			},
			config: &test,
			expected: Client{
				etcdClient: etcd.NewClient([]string{
					"http://127.0.0.1:4001",
					"http://127.0.0.1:4002",
					"http://127.0.0.1:4003",
				}),
				config: reflect.ValueOf(&test),
				info: map[string]info{
					"/":       info{field: reflect.ValueOf(&test).Elem()},
					"/field2": info{field: reflect.ValueOf(&test.Field2).Elem()},
				},
			},
		},
		{
			description: "it should fail to preload a non-pointer structure",
			config:      test,
			expectedErr: true,
		},
		{
			description: "it should deny a non-pointer to structure",
			config:      struct{}{},
			expectedErr: true,
		},
		{
			description: "it should deny a pointer to something that is not a structure",
			config:      &[]int{},
			expectedErr: true,
		},
	}

	for i, item := range data {
		c, err := NewClient(item.machines, item.config)
		if err == nil && item.expectedErr {
			t.Errorf("Item %d, “%s”: error expected", i, item.description)
			continue

		} else if err != nil && !item.expectedErr {
			t.Errorf("Item %d, “%s”: unexpected error. %s", i, item.description, err.Error())
			continue
		}

		if !item.expectedErr && !equalClients(c, &item.expected) {
			t.Errorf("Item %d, “%s”: objects mismatch. Expecting “%+v”; found “%+v”",
				i, item.description, item.expected, c)
		}
	}
}

func TestSave(t *testing.T) {
	data := []struct {
		description string            // describe the test case
		init        func(*clientMock) // initial configuration of the mocked client (if necessary)
		config      interface{}       // configuration instance (structure) to save
		expectedErr bool              // error expectation when saving the configuration
		expected    etcd.Node         // etcd state after saving the configuration (only when there's no error)
	}{
		{
			description: "it should save an one-level configuration pointer ignoring not tagged fields",
			config: &struct {
				Field1 string `etcd:"/field1"`
				Field2 int    `etcd:"/field2"`
				Field3 int64  `etcd:"/field3"`
				Field4 bool   `etcd:"/field4"`
				Extra  string
			}{
				Field1: "value1",
				Field2: 10,
				Field3: 20,
				Field4: true,
				Extra:  "shouldn't be saved",
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field1",
						Value: "value1",
					},
					{
						Key:   "/field2",
						Value: "10",
					},
					{
						Key:   "/field3",
						Value: "20",
					},
					{
						Key:   "/field4",
						Value: "true",
					},
				},
			},
		},
		{
			description: "it should save an embedded structure",
			config: struct {
				Field struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{
				Field: struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				}{
					Subfield1: 10,
					Subfield2: 20,
					Subfield3: false,
				},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/subfield1",
								Value: "10",
							},
							{
								Key:   "/field/subfield2",
								Value: "20",
							},
							{
								Key:   "/field/subfield3",
								Value: "false",
							},
						},
					},
				},
			},
		},
		{
			description: "it should save a slice of strings",
			config: struct {
				Field []string `etcd:"/field"`
			}{
				Field: []string{"value1", "value2", "value3"},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "value1",
							},
							{
								Key:   "/field/1",
								Value: "value2",
							},
							{
								Key:   "/field/2",
								Value: "value3",
							},
						},
					},
				},
			},
		},
		{
			description: "it should save a slice of structures",
			config: struct {
				Field []struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{
				Field: []struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				}{
					{
						Subfield1: 10,
						Subfield2: 20,
						Subfield3: false,
					},
					{
						Subfield1: 20,
						Subfield2: 40,
						Subfield3: true,
					},
					{
						Subfield1: 40,
						Subfield2: 80,
						Subfield3: false,
					},
				},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key: "/field/0",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/0/subfield1",
										Value: "10",
									},
									{
										Key:   "/field/0/subfield2",
										Value: "20",
									},
									{
										Key:   "/field/0/subfield3",
										Value: "false",
									},
								},
							},
							{
								Key: "/field/1",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/1/subfield1",
										Value: "20",
									},
									{
										Key:   "/field/1/subfield2",
										Value: "40",
									},
									{
										Key:   "/field/1/subfield3",
										Value: "true",
									},
								},
							},
							{
								Key: "/field/2",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/2/subfield1",
										Value: "40",
									},
									{
										Key:   "/field/2/subfield2",
										Value: "80",
									},
									{
										Key:   "/field/2/subfield3",
										Value: "false",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "it should save a map of string to string",
			config: struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: map[string]string{
					"subfield1": "value1",
					"subfield2": "value2",
					"subfield3": "value3",
				},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/subfield1",
								Value: "value1",
							},
							{
								Key:   "/field/subfield2",
								Value: "value2",
							},
							{
								Key:   "/field/subfield3",
								Value: "value3",
							},
						},
					},
				},
			},
		},
		{
			description: "it should fail to save a non-structure",
			config:      123,
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a set string",
			init: func(c *clientMock) {
				c.setErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field string `etcd:"/field"`
			}{
				Field: "value",
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a set int",
			init: func(c *clientMock) {
				c.setErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field int `etcd:"/field"`
			}{
				Field: 10,
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a set int64",
			init: func(c *clientMock) {
				c.setErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field int64 `etcd:"/field"`
			}{
				Field: 20,
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a set bool",
			init: func(c *clientMock) {
				c.setErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field bool `etcd:"/field"`
			}{
				Field: true,
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a set struct",
			init: func(c *clientMock) {
				c.setErrors["/field/subfield"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{
				Field: struct {
					Subfield int `etcd:"/subfield"`
				}{
					Subfield: 10,
				},
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to create the slice path with an unknown error",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = fmt.Errorf("generic error")
			},
			config: struct {
				Field []string `etcd:"/field"`
			}{
				Field: []string{"value"},
			},
			expectedErr: true,
		},
		{
			description: "it should save when etcd rejects to create the slice path because it already exists",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeNodeExist)}
			},
			config: struct {
				Field []string `etcd:"/field"`
			}{
				Field: []string{"value"},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "value",
							},
						},
					},
				},
			},
		},
		{
			description: "it should fail when etcd rejects to create the slice path",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field []string `etcd:"/field"`
			}{
				Field: []string{"value"},
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to create the index path for the structure",
			init: func(c *clientMock) {
				c.createDirErrors["/field/0"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field []struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{
				Field: []struct {
					Subfield int `etcd:"/subfield"`
				}{
					{
						Subfield: 10,
					},
				},
			},
			expectedErr: true,
		},
		{
			description: "it should fails when etcd rejects to create the index path with an unknown error",
			init: func(c *clientMock) {
				c.createDirErrors["/field/0"] = fmt.Errorf("generic error")
			},
			config: struct {
				Field []struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{
				Field: []struct {
					Subfield int `etcd:"/subfield"`
				}{
					{
						Subfield: 10,
					},
				},
			},
			expectedErr: true,
		},
		{
			description: "it should save when etcd rejects to create the index path for the structure because it already exists",
			init: func(c *clientMock) {
				c.createDirErrors["/field/0"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeNodeExist)}
			},
			config: struct {
				Field []struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{
				Field: []struct {
					Subfield int `etcd:"/subfield"`
				}{
					{
						Subfield: 10,
					},
				},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key: "/field/0",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/0/subfield",
										Value: "10",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "it should fail when etcd rejects a slice of struct values",
			init: func(c *clientMock) {
				c.setErrors["/field/0/subfield"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field []struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{
				Field: []struct {
					Subfield int `etcd:"/subfield"`
				}{
					{
						Subfield: 10,
					},
				},
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a slice of string values",
			init: func(c *clientMock) {
				c.createInOrderErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field []string `etcd:"/field"`
			}{
				Field: []string{"value"},
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects creating the path that stores the map values",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: map[string]string{
					"subfield": "value",
				},
			},
			expectedErr: true,
		},
		{
			description: "it should fails when etcd rejects to create the map path with an unknown error",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = fmt.Errorf("generic error")
			},
			config: struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: map[string]string{
					"subfield": "value",
				},
			},
			expectedErr: true,
		},
		{
			description: "it should save when etcd rejects to create the map path that stores the map values because it already exists it",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeNodeExist)}
			},
			config: struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: map[string]string{
					"subfield": "value",
				},
			},
			expected: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/subfield",
								Value: "value",
							},
						},
					},
				},
			},
		},
		{
			description: "it should fail when etcd rejects a set map",
			init: func(c *clientMock) {
				c.setErrors["/field/subfield"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: map[string]string{
					"subfield": "value",
				},
			},
			expectedErr: true,
		},
	}

	for i, item := range data {
		if DEBUG {
			fmt.Printf(">>> Running TestSave for index %d\n", i)
		}

		mock := NewClientMock()
		c := Client{
			etcdClient: mock,
			config:     reflect.ValueOf(item.config),
			info:       make(map[string]info),
		}

		if item.init != nil {
			item.init(mock)
		}

		err := c.Save()
		if err == nil && item.expectedErr {
			t.Errorf("Item %d, “%s”: error expected", i, item.description)
			continue

		} else if err != nil && !item.expectedErr {
			t.Errorf("Item %d, “%s”: unexpected error. %s", i, item.description, err.Error())
			continue
		}

		if !item.expectedErr && !equalNodes(mock.root, &item.expected) {
			t.Errorf("Item %d, “%s”: nodes mismatch. Expecting “%s”; found “%s”",
				i, item.description, printNode(&item.expected), printNode(mock.root))
		}
	}
}

func BenchmarkSave(b *testing.B) {
	mock := NewClientMock()
	c := Client{
		etcdClient: mock,
		config: reflect.ValueOf(struct {
			Field string `etcd:"field"`
		}{
			Field: "value",
		}),
		info: make(map[string]info),
	}

	for i := 0; i < b.N; i++ {
		if err := c.Save(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLoad(t *testing.T) {
	data := []struct {
		description string            // describe the test case
		init        func(*clientMock) // initial configuration of the mocked client (if necessary)
		etcdData    etcd.Node         // etcd state before loading the configuration
		config      interface{}       // configuration structure (used to detect what we need to look for in etcd)
		expectedErr bool              // error expectation when loading the configuration
		expected    interface{}       // configuration instance expected after loading
	}{
		{
			description: "it should load an one-level configuration ignoring not tagged fields",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field1",
						Value: "value1",
					},
					{
						Key:   "/field2",
						Value: "10",
					},
					{
						Key:   "/field3",
						Value: "20",
					},
					{
						Key:   "/field4",
						Value: "true",
					},
				},
			},
			config: &struct {
				Field1 string `etcd:"/field1"`
				Field2 int    `etcd:"/field2"`
				Field3 int64  `etcd:"/field3"`
				Field4 bool   `etcd:"/field4"`
				Extra  string
			}{},
			expected: struct {
				Field1 string `etcd:"/field1"`
				Field2 int    `etcd:"/field2"`
				Field3 int64  `etcd:"/field3"`
				Field4 bool   `etcd:"/field4"`
				Extra  string
			}{
				Field1: "value1",
				Field2: 10,
				Field3: 20,
				Field4: true,
			},
		},
		{
			description: "it should load an embedded structure",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/subfield1",
								Value: "10",
							},
							{
								Key:   "/field/subfield2",
								Value: "20",
							},
							{
								Key:   "/field/subfield3",
								Value: "false",
							},
						},
					},
				},
			},
			config: &struct {
				Field1 struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{},
			expected: struct {
				Field1 struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{
				Field1: struct {
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				}{
					Subfield1: 10,
					Subfield2: 20,
					Subfield3: false,
				},
			},
		},
		{
			description: "it should load a slice of strings",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "value1",
							},
							{
								Key:   "/field/1",
								Value: "value2",
							},
							{
								Key:   "/field/2",
								Value: "value3",
							},
						},
					},
				},
			},
			config: &struct {
				Field []string `etcd:"/field"`
			}{},
			expected: struct {
				Field []string `etcd:"/field"`
			}{
				Field: []string{"value1", "value2", "value3"},
			},
		},
		{
			description: "it should load a slice of int",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "10",
							},
							{
								Key:   "/field/1",
								Value: "20",
							},
							{
								Key:   "/field/2",
								Value: "30",
							},
						},
					},
				},
			},
			config: &struct {
				Field []int `etcd:"/field"`
			}{},
			expected: struct {
				Field []int `etcd:"/field"`
			}{
				Field: []int{10, 20, 30},
			},
		},
		{
			description: "it should load a slice of int64",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "10",
							},
							{
								Key:   "/field/1",
								Value: "20",
							},
							{
								Key:   "/field/2",
								Value: "30",
							},
						},
					},
				},
			},
			config: &struct {
				Field []int64 `etcd:"/field"`
			}{},
			expected: struct {
				Field []int64 `etcd:"/field"`
			}{
				Field: []int64{10, 20, 30},
			},
		},
		{
			description: "it should load a slice of bool",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "true",
							},
							{
								Key:   "/field/1",
								Value: "false",
							},
							{
								Key:   "/field/2",
								Value: "true",
							},
						},
					},
				},
			},
			config: &struct {
				Field []bool `etcd:"/field"`
			}{},
			expected: struct {
				Field []bool `etcd:"/field"`
			}{
				Field: []bool{true, false, true},
			},
		},
		{
			description: "it should load a slice of structures",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key: "/field/0",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/0/subfield1",
										Value: "10",
									},
									{
										Key:   "/field/0/subfield2",
										Value: "20",
									},
									{
										Key:   "/field/0/subfield3",
										Value: "false",
									},
								},
							},
							{
								Key: "/field/1",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/1/subfield1",
										Value: "20",
									},
									{
										Key:   "/field/1/subfield2",
										Value: "40",
									},
									{
										Key:   "/field/1/subfield3",
										Value: "true",
									},
								},
							},
							{
								Key: "/field/2",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/2/subfield1",
										Value: "40",
									},
									{
										Key:   "/field/2/subfield2",
										Value: "80",
									},
									{
										Key:   "/field/2/subfield3",
										Value: "false",
									},
								},
							},
						},
					},
				},
			},
			config: &struct {
				Field []struct {
					Subfield0 string
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{},
			expected: struct {
				Field []struct {
					Subfield0 string
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{
				Field: []struct {
					Subfield0 string
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				}{
					{
						Subfield1: 10,
						Subfield2: 20,
						Subfield3: false,
					},
					{
						Subfield1: 20,
						Subfield2: 40,
						Subfield3: true,
					},
					{
						Subfield1: 40,
						Subfield2: 80,
						Subfield3: false,
					},
				},
			},
		},
		{
			description: "it should save a map of string to string",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/subfield1",
								Value: "value1",
							},
							{
								Key:   "/field/subfield2",
								Value: "value2",
							},
							{
								Key:   "/field/subfield3",
								Value: "value3",
							},
						},
					},
				},
			},
			config: &struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: make(map[string]string),
			},
			expected: struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: map[string]string{
					"subfield1": "value1",
					"subfield2": "value2",
					"subfield3": "value3",
				},
			},
		},
		{
			description: "it should fail to load a non-pointer to structure",
			config:      123,
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a get string",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field string `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a get int",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field int `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a get int64",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field int64 `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd returns a number with an invalid format",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field",
						Value: "NaN",
					},
				},
			},
			config: &struct {
				Field int `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects a get bool",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field bool `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get a structure field",
			init: func(c *clientMock) {
				c.getErrors["/field/subfield"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get a slice of structure",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field []struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd returns corrupted data in a structure of a slice",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key: "/field/0",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/0/subfield",
										Value: "NaN",
									},
								},
							},
						},
					},
				},
			},
			config: &struct {
				Field []struct {
					Subfield int `etcd:"/subfield"`
				} `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get the slice of strings",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field []string `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get the slice of int",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field []int `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd returns an invalid int in a slice",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "NaN",
							},
						},
					},
				},
			},
			config: &struct {
				Field []int `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get the slice of int64",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field []int64 `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd returns an invalid int64 in a slice",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/0",
								Value: "NaN",
							},
						},
					},
				},
			},
			config: &struct {
				Field []int64 `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get the slice of bool",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field []bool `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when trying to load into a nil map",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key:   "/field/subfield1",
								Value: "value1",
							},
							{
								Key:   "/field/subfield2",
								Value: "value2",
							},
							{
								Key:   "/field/subfield3",
								Value: "value3",
							},
						},
					},
				},
			},
			config: &struct {
				Field map[string]string `etcd:"/field"`
			}{},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd rejects to get a map",
			init: func(c *clientMock) {
				c.getErrors["/field"] = &etcd.EtcdError{ErrorCode: int(etcdErrorCodeRaftInternal)}
			},
			config: &struct {
				Field map[string]string `etcd:"/field"`
			}{
				Field: make(map[string]string),
			},
			expectedErr: true,
		},
		{
			description: "it should fail when etcd data is corrupted",
			etcdData: etcd.Node{
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key: "/field",
						Dir: true,
						Nodes: etcd.Nodes{
							{
								Key: "/field/subfield",
								Dir: true,
								Nodes: etcd.Nodes{
									{
										Key:   "/field/subfield/subsubfield2",
										Value: "NaN",
									},
								},
							},
						},
					},
				},
			},
			config: &struct {
				Field struct {
					Subfield struct {
						Subsubfield1 string
						Subsubfield2 int `etcd:"/subsubfield2"`
					} `etcd:"/subfield"`
				} `etcd:"/field"`
			}{},
			expectedErr: true,
		},
	}

	for i, item := range data {
		if DEBUG {
			fmt.Printf(">>> Running TestLoad for index %d\n", i)
		}

		mock := NewClientMock()
		mock.root = &item.etcdData

		c := Client{
			etcdClient: mock,
			config:     reflect.ValueOf(item.config),
			info:       make(map[string]info),
		}

		if item.init != nil {
			item.init(mock)
		}

		err := c.Load()
		if err == nil && item.expectedErr {
			t.Errorf("Item %d, “%s”: error expected", i, item.description)
			continue

		} else if err != nil && !item.expectedErr {
			t.Errorf("Item %d, “%s”: unexpected error. %s", i, item.description, err.Error())
			continue
		}

		if !item.expectedErr && reflect.DeepEqual(item.config, item.expected) {
			t.Errorf("Item %d, “%s”: config mismatch. Expecting “%+v”; found “%+v”",
				i, item.description, item.expected, item.config)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	mock := NewClientMock()
	mock.root = &etcd.Node{
		Dir: true,
		Nodes: etcd.Nodes{
			{
				Key:   "/field",
				Value: "value",
			},
		},
	}

	c := Client{
		etcdClient: mock,
		config: reflect.ValueOf(&struct {
			Field string `etcd:"field"`
		}{}),
		info: make(map[string]info),
	}

	for i := 0; i < b.N; i++ {
		if err := c.Load(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestWatch(t *testing.T) {
	config := struct {
		Field1  string            `etcd:"/field1"`
		Field2  int               `etcd:"/field2"`
		Field3  int64             `etcd:"/field3"`
		Field4  bool              `etcd:"/field4"`
		Field5  map[string]string `etcd:"/field5"`
		Field6  []string          `etcd:"/field6"`
		Field7  []int             `etcd:"/field7"`
		Field8  []int64           `etcd:"/field8"`
		Field9  []bool            `etcd:"/field9"`
		Field10 struct {
			Subfield1 string `etcd:"/subfield1"`
			Subfield2 int    `etcd:"/subfield2"`
			Subfield3 int64  `etcd:"/subfield3"`
			Subfield4 bool   `etcd:"/subfield4"`
		} `etcd:"/field10"`
	}{
		Field5: make(map[string]string),
	}

	etcdData := etcd.Node{
		Dir: true,
		Nodes: etcd.Nodes{
			{
				Key:   "/field1",
				Value: "value1",
			},
			{
				Key:   "/field2",
				Value: "10",
			},
			{
				Key:   "/field3",
				Value: "20",
			},
			{
				Key:   "/field4",
				Value: "true",
			},
			{
				Key: "/field5",
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field5/subfield1",
						Value: "subvalue1",
					},
				},
			},
			{
				Key: "/field6",
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field6/0",
						Value: "subvalue1",
					},
					{
						Key:   "/field6/1",
						Value: "subvalue2",
					},
				},
			},
			{
				Key: "/field7",
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field7/0",
						Value: "100",
					},
					{
						Key:   "/field7/1",
						Value: "200",
					},
				},
			},
			{
				Key: "/field8",
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field8/0",
						Value: "1000",
					},
					{
						Key:   "/field8/1",
						Value: "2000",
					},
				},
			},
			{
				Key: "/field9",
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field9/0",
						Value: "true",
					},
					{
						Key:   "/field9/1",
						Value: "false",
					},
				},
			},
			{
				Key: "/field10",
				Dir: true,
				Nodes: etcd.Nodes{
					{
						Key:   "/field10/subfield1",
						Value: "subvalue1",
					},
					{
						Key:   "/field10/subfield2",
						Value: "500",
					},
					{
						Key:   "/field10/subfield3",
						Value: "800",
					},
					{
						Key:   "/field10/subfield4",
						Value: "true",
					},
				},
			},
		},
	}

	data := []struct {
		description string            // describe the test case
		init        func(*clientMock) // initial configuration of the mocked client (if necessary)
		field       interface{}       // field that will be monitored for changes
		changeValue etcd.Node         // value injected in the change
		expectedErr bool              // error expectation when watching the configuration
		expected    interface{}       // value expected in the field after the callback is called
	}{
		{
			description: "it should watch a string field",
			field:       &config.Field1,
			changeValue: etcd.Node{
				Value: "value1 modified",
			},
			expected: "value1 modified",
		},
		{
			description: "it should watch an int field",
			field:       &config.Field2,
			changeValue: etcd.Node{
				Value: "13",
			},
			expected: int(13),
		},
		{
			description: "it should watch an int64 field",
			field:       &config.Field3,
			changeValue: etcd.Node{
				Value: "27",
			},
			expected: int64(27),
		},
		{
			description: "it should watch a bool field",
			field:       &config.Field4,
			changeValue: etcd.Node{
				Value: "false",
			},
			expected: false,
		},
		{
			description: "it should watch a map field",
			field:       &config.Field5,
			changeValue: etcd.Node{
				Nodes: etcd.Nodes{
					{
						Key:   "/field5/subfield2",
						Value: "subvalue2 modified",
					},
					{
						Key:   "/field5/subfield3",
						Value: "subvalue3 modified",
					},
				},
			},
			expected: map[string]string{
				"subfield2": "subvalue2 modified",
				"subfield3": "subvalue3 modified",
			},
		},
		{
			description: "it should watch a slice of strings",
			field:       &config.Field6,
			changeValue: etcd.Node{
				Nodes: etcd.Nodes{
					{
						Key:   "/field6/0",
						Value: "subvalue1 modified",
					},
					{
						Key:   "/field6/1",
						Value: "subvalue2 modified",
					},
				},
			},
			expected: []string{
				"subvalue1 modified",
				"subvalue2 modified",
			},
		},
		{
			description: "it should watch a slice of int",
			field:       &config.Field7,
			changeValue: etcd.Node{
				Nodes: etcd.Nodes{
					{
						Key:   "/field7/0",
						Value: "133",
					},
					{
						Key:   "/field7/1",
						Value: "212",
					},
				},
			},
			expected: []int{133, 212},
		},
		{
			description: "it should watch a slice of int64",
			field:       &config.Field8,
			changeValue: etcd.Node{
				Nodes: etcd.Nodes{
					{
						Key:   "/field8/0",
						Value: "1486",
					},
					{
						Key:   "/field8/1",
						Value: "2950",
					},
				},
			},
			expected: []int64{1486, 2950},
		},
		{
			description: "it should watch a slice of bool",
			field:       &config.Field9,
			changeValue: etcd.Node{
				Nodes: etcd.Nodes{},
			},
			expected: []bool{},
		},
		{
			description: "it should watch a structure",
			field:       &config.Field10,
			changeValue: etcd.Node{
				Nodes: etcd.Nodes{
					{
						Key:   "/field10/subfield1",
						Value: "subvalue1 modified",
					},
					{
						Key:   "/field10/subfield2",
						Value: "529",
					},
					{
						Key:   "/field10/subfield3",
						Value: "861",
					},
					{
						Key:   "/field10/subfield4",
						Value: "false",
					},
				},
			},
			expected: struct {
				Subfield1 string `etcd:"/subfield1"`
				Subfield2 int    `etcd:"/subfield2"`
				Subfield3 int64  `etcd:"/subfield3"`
				Subfield4 bool   `etcd:"/subfield4"`
			}{
				Subfield1: "subvalue1 modified",
				Subfield2: int(529),
				Subfield3: int64(861),
				Subfield4: false,
			},
		},
		{
			description: "it should fail when watching an invalid field",
			field:       "I'm not a valid field",
			expectedErr: true,
		},
		{
			description: "it should fail when watching a field not registered before",
			field:       &struct{}{},
			expectedErr: true,
		},
	}

	for i, item := range data {
		if DEBUG {
			fmt.Printf(">>> Running TestWatch for index %d\n", i)
		}

		mock := NewClientMock()
		mock.root = &etcdData

		c := Client{
			etcdClient: mock,
			config:     reflect.ValueOf(&config),
			info:       make(map[string]info),
		}

		if item.init != nil {
			item.init(mock)
		}

		c.preload(c.config, "")

		done := make(chan bool)
		stop, err := c.Watch(item.field, func() {
			done <- true
		})

		if err == nil && item.expectedErr {
			t.Errorf("Item %d, “%s”: error expected", i, item.description)
			continue

		} else if err != nil && !item.expectedErr {
			t.Errorf("Item %d, “%s”: unexpected error. %s", i, item.description, err.Error())
			continue
		}

		if err != nil {
			continue
		}

		mock.notifyChange(item.changeValue)
		<-done
		close(stop)

		value := reflect.ValueOf(item.field)
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}

		if !reflect.DeepEqual(value.Interface(), item.expected) {
			t.Errorf("Item %d, “%s”: fields mismatch. Expecting “%+v”; found “%+v”",
				i, item.description, item.expected, item.field)
		}
	}
}

func BenchmarkWatch(b *testing.B) {
	mock := NewClientMock()
	mock.root = &etcd.Node{
		Dir: true,
		Nodes: etcd.Nodes{
			{
				Key:   "/field",
				Value: "value",
			},
		},
	}

	s := struct {
		Field string `etcd:"field"`
	}{}

	c := Client{
		etcdClient: mock,
		config:     reflect.ValueOf(&s),
		info:       make(map[string]info),
	}

	c.preload(c.config, "")

	called := make(chan bool)
	for i := 0; i < b.N; i++ {
		stop, err := c.Watch(&s.Field, func() {
			called <- true
		})

		if err != nil {
			b.Fatal(err)
		}

		mock.notifyChange(etcd.Node{
			Value: "abc",
		})

		select {
		case <-called:
			close(stop)
		}
	}
}

//////////////////////////////////////
//////////////////////////////////////
//////////////////////////////////////

type clientMock struct {
	root      *etcd.Node     // root node
	etcdIndex uint64         // control update sequence
	change    chan etcd.Node // simulate config changes for watch

	// force errors for specific methods and paths
	createDirErrors     map[string]error
	createInOrderErrors map[string]error
	setErrors           map[string]error
	getErrors           map[string]error
	watchErrors         map[string]error
}

func NewClientMock() *clientMock {
	return &clientMock{
		root: &etcd.Node{
			Dir: true,
		},
		change:              make(chan etcd.Node),
		createDirErrors:     make(map[string]error),
		createInOrderErrors: make(map[string]error),
		setErrors:           make(map[string]error),
		getErrors:           make(map[string]error),
		watchErrors:         make(map[string]error),
	}
}

func (c *clientMock) CreateDir(path string, ttl uint64) (*etcd.Response, error) {
	if DEBUG {
		fmt.Printf(" - Creating path %s\n", path)
	}

	// CreatDir error is a special case, because we could have the "already created" error
	err := c.createDirErrors[path]
	if etcderr, ok := err.(*etcd.EtcdError); ok && etcderr.ErrorCode != int(etcdErrorCodeNodeExist) {
		return nil, err
	}

	c.etcdIndex++
	current := c.createDirsInPath(path, ttl)

	parts := strings.Split(path, "/")
	found := false

	for _, n := range current.Nodes {
		if n.Key == parts[len(parts)-1] {
			found = true
			current = n
			break
		}
	}

	if !found {
		if DEBUG {
			fmt.Printf("  > Directory %s created\n", path)
		}

		newNode := &etcd.Node{
			Key:           path,
			Dir:           true,
			TTL:           int64(ttl),
			ModifiedIndex: c.etcdIndex,
			CreatedIndex:  c.etcdIndex,
		}

		current.Nodes = append(current.Nodes, newNode)
		current = newNode
	}

	return &etcd.Response{
		Action:    "create",
		Node:      current,
		EtcdIndex: c.etcdIndex,
	}, err
}

func (c *clientMock) CreateInOrder(path string, value string, ttl uint64) (*etcd.Response, error) {
	if DEBUG {
		fmt.Printf(" - Creating in order path %s with value “%s”\n", path, value)
	}

	if err := c.createInOrderErrors[path]; err != nil {
		return nil, err
	}

	c.etcdIndex++
	current := c.createDirsInPath(path, ttl)

	for _, n := range current.Nodes {
		if n.Key == path {
			current = n
			break
		}
	}

	path = path + "/" + strconv.Itoa(len(current.Nodes))

	if DEBUG {
		fmt.Printf("  > Key %s created\n", path)
	}

	newNode := &etcd.Node{
		Key:           path,
		Value:         value,
		TTL:           int64(ttl),
		ModifiedIndex: c.etcdIndex,
		CreatedIndex:  c.etcdIndex,
	}
	current.Nodes = append(current.Nodes, newNode)

	return &etcd.Response{
		Action:    "create",
		Node:      newNode,
		EtcdIndex: c.etcdIndex,
	}, nil
}

func (c *clientMock) Set(path string, value string, ttl uint64) (*etcd.Response, error) {
	if DEBUG {
		fmt.Printf(" - Setting path %s with value “%s”\n", path, value)
	}

	if err := c.setErrors[path]; err != nil {
		return nil, err
	}

	c.etcdIndex++
	current := c.createDirsInPath(path, ttl)

	found := false
	for _, n := range current.Nodes {
		if n.Key == path {
			if n.Dir {
				return nil, &etcd.EtcdError{ErrorCode: int(etcdErrorCodeNotFile), Message: path}
			}

			found = true
			current = n
			break
		}
	}

	var oldNode *etcd.Node
	var action string

	if found {
		if DEBUG {
			fmt.Printf("  > Key %s updated\n", path)
		}

		oldNode = new(etcd.Node)
		*oldNode = *current

		current.Value = value
		current.TTL = int64(ttl)
		current.ModifiedIndex = c.etcdIndex
		action = "update"

	} else {
		if DEBUG {
			fmt.Printf("  > Key %s created\n", path)
		}

		newNode := &etcd.Node{
			Key:           path,
			Value:         value,
			TTL:           int64(ttl),
			ModifiedIndex: c.etcdIndex,
			CreatedIndex:  c.etcdIndex,
		}
		current.Nodes = append(current.Nodes, newNode)
		current = newNode
		action = "create"
	}

	return &etcd.Response{
		Action:    action,
		Node:      current,
		PrevNode:  oldNode,
		EtcdIndex: c.etcdIndex,
	}, nil
}

func (c *clientMock) Get(path string, sort, recursive bool) (*etcd.Response, error) {
	if DEBUG {
		fmt.Printf(" - Getting path %s\n", path)
	}

	if err := c.getErrors[path]; err != nil {
		return nil, err
	}

	current := c.root
	currentPath := c.root.Key
	parts := strings.Split(path, "/")

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		currentPath += "/" + part

		found := false
		for _, n := range current.Nodes {
			if n.Key == currentPath {
				found = true
				current = n
				break
			}
		}

		if !found {
			return nil, &etcd.EtcdError{ErrorCode: int(etcdErrorCodeKeyNotFound), Message: path}
		}
	}

	return &etcd.Response{
		Action:    "get",
		Node:      current,
		EtcdIndex: c.etcdIndex,
	}, nil
}

func (c *clientMock) Watch(
	path string,
	waitIndex uint64,
	recursive bool,
	receiver chan *etcd.Response,
	stop chan bool,
) (*etcd.Response, error) {

	if DEBUG {
		fmt.Printf(" - Watching path %s\n", path)
	}

	if err := c.watchErrors[path]; err != nil {
		return nil, err
	}

	current := c.root
	currentPath := c.root.Key
	parts := strings.Split(path, "/")

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		currentPath += "/" + part

		found := false
		for _, n := range current.Nodes {
			if n.Key == currentPath {
				found = true
				current = n
				break
			}
		}

		if !found {
			return nil, &etcd.EtcdError{ErrorCode: int(etcdErrorCodeKeyNotFound), Message: path}
		}
	}

	select {
	case node := <-c.change:
		current.Value = node.Value
		current.Nodes = node.Nodes

		receiver <- &etcd.Response{
			Action:    "get",
			Node:      current,
			EtcdIndex: c.etcdIndex,
		}
	case <-stop:
	}

	return nil, nil
}

func (c *clientMock) createDirsInPath(path string, ttl uint64) *etcd.Node {
	if DEBUG {
		fmt.Printf("  > Creating parent paths %s\n", path)
	}

	current := c.root
	currentPath := c.root.Key
	parts := strings.Split(path, "/")

	// We ignore the first and last index, because we already have the root and don't know what to do
	// with the last part of the path (dir or key)
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]
		currentPath += "/" + part

		found := false
		for _, n := range current.Nodes {
			if n.Key == currentPath {
				found = true
				current = n
				break
			}
		}

		if found {
			continue
		}

		if DEBUG {
			fmt.Printf("   ... Directory %s created (parent path)\n", currentPath)
		}

		newNode := &etcd.Node{
			Key:           currentPath,
			Dir:           true,
			TTL:           int64(ttl),
			ModifiedIndex: c.etcdIndex,
			CreatedIndex:  c.etcdIndex,
		}

		current.Nodes = append(current.Nodes, newNode)
		current = newNode
	}

	return current
}

func (c *clientMock) notifyChange(node etcd.Node) {
	c.etcdIndex++
	node.ModifiedIndex = c.etcdIndex
	// TODO: Modify all children nodes versions
	c.change <- node
}

func equalClients(c1, c2 *Client) bool {
	if c1.config != c2.config ||
		(c1.etcdClient == nil && c2.etcdClient != nil) ||
		(c1.etcdClient != nil && c2.etcdClient == nil) {

		return false
	}

	for path1, value1 := range c1.info {
		found := false
		for path2, value2 := range c2.info {
			if path1 == path2 {
				found = true
				if !reflect.DeepEqual(value1, value2) {
					return false
				}
				break
			}
		}

		if !found {
			return false
		}
	}

	for path2 := range c2.info {
		found := false
		for path1 := range c1.info {
			if path1 == path2 {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func equalNodes(n1, n2 *etcd.Node) bool {
	if n1.Key != n2.Key ||
		n1.Value != n2.Value ||
		n1.Dir != n2.Dir ||
		n1.TTL != n2.TTL ||
		len(n1.Nodes) != len(n2.Nodes) {

		return false
	}

	// Children are not ordered
	for _, c1 := range n1.Nodes {
		foundEqual := false
		for _, c2 := range n2.Nodes {
			if equalNodes(c1, c2) {
				foundEqual = true
				break
			}
		}

		if !foundEqual {
			return false
		}
	}

	return true
}

func printNode(n *etcd.Node) string {
	if n == nil {
		return ""
	}

	dir := "false"
	if n.Dir {
		dir = "true"
	}

	ttl := strconv.FormatInt(n.TTL, 10)

	output := "{ " +
		"Key: '" + n.Key + "', " +
		"Value: '" + n.Value + "', " +
		"Dir: " + dir + ", " +
		"TTL: " + ttl + ", " +
		"Nodes: ["

	for _, c := range n.Nodes {
		output += printNode(c)
	}

	output += "] }"
	return output
}
