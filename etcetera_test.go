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

	etcderrors "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

const DEBUG = false

func ExampleSaveLoad() {
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

	a1 := A{
		Field1: "value1",
		Field2: 10,
		Field3: 999,
		Field4: true,
		Field5: B{"value2"},
		Field6: map[string]string{"key1": "value3"},
		Field7: []string{"value4", "value5", "value6"},
	}

	client := etcd.NewClient([]string{
		"http://127.0.0.1:4001",
	})

	if err := Save(&a1, client); err != nil {
		fmt.Println(err.Error())
		return
	}

	a2 := A{
		Field6: make(map[string]string),
	}

	if err := Load(&a2, client); err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("Input: %+v\n", a1)
	fmt.Printf("Output: %+v\n", a2)
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
				c.setErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.setErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.setErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.setErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.setErrors["/field/subfield"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
			description: "it should fail when etcd rejects to create the slice path",
			init: func(c *clientMock) {
				c.createDirErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.createDirErrors["/field"] = &etcd.EtcdError{ErrorCode: etcderrors.EcodeNodeExist}
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
			description: "it should fail when etcd rejects to create the index path for the structure",
			init: func(c *clientMock) {
				c.createDirErrors["/field/0"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.createDirErrors["/field/0"] = &etcd.EtcdError{ErrorCode: etcderrors.EcodeNodeExist}
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
				c.setErrors["/field/0/subfield"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.createInOrderErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.createDirErrors["/field"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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
				c.createDirErrors["/field"] = &etcd.EtcdError{ErrorCode: etcderrors.EcodeNodeExist}
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
				c.setErrors["/field/subfield"] = etcderrors.NewRequestError(etcderrors.EcodeRaftInternal, "")
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

		c := NewClientMock()

		if item.init != nil {
			item.init(c)
		}

		err := Save(item.config, c)
		if err == nil && item.expectedErr {
			t.Errorf("Item %d, “%s”: error expected", i, item.description)
			continue

		} else if err != nil && !item.expectedErr {
			t.Errorf("Item %d, “%s”: unexpected error. %s", i, item.description, err.Error())
			continue
		}

		if !item.expectedErr && !equalNodes(c.root, &item.expected) {
			t.Errorf("Item %d, “%s”: nodes mismatch. Expecting “%s”; found “%s”",
				i, item.description, printNode(&item.expected), printNode(c.root))
		}
	}
}

func TestLoad(t *testing.T) {
	data := []struct {
		description string      // describe the test case
		etcdData    etcd.Node   // etcd state before loading the configuration
		config      interface{} // configuration structure (used to detect what we need to look for in etcd)
		expectedErr bool        // error expectation when loading the configuration
		expected    interface{} // configuration instance expected after loading
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
					Subfield1 int   `etcd:"/subfield1"`
					Subfield2 int64 `etcd:"/subfield2"`
					Subfield3 bool  `etcd:"/subfield3"`
				} `etcd:"/field"`
			}{},
			expected: struct {
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
	}

	for i, item := range data {
		if DEBUG {
			fmt.Printf(">>> Running TestLoad for index %d\n", i)
		}

		c := NewClientMock()
		c.root = &item.etcdData

		err := Load(item.config, c)
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

//////////////////////////////////////
//////////////////////////////////////
//////////////////////////////////////

type clientMock struct {
	root      *etcd.Node // root node
	etcdIndex uint64     // control update sequence

	// force errors for specific methods and paths
	createDirErrors     map[string]error
	createInOrderErrors map[string]error
	setErrors           map[string]error
	getErrors           map[string]error
}

func NewClientMock() *clientMock {
	return &clientMock{
		root: &etcd.Node{
			Dir: true,
		},
		createDirErrors:     make(map[string]error),
		createInOrderErrors: make(map[string]error),
		setErrors:           make(map[string]error),
		getErrors:           make(map[string]error),
	}
}

func (c *clientMock) CreateDir(path string, ttl uint64) (*etcd.Response, error) {
	if DEBUG {
		fmt.Printf(" - Creating path %s\n", path)
	}

	// CreatDir error is a special case, because we could have the "already created" error
	err := c.createDirErrors[path]
	if etcderr, ok := err.(*etcd.EtcdError); ok && etcderr.ErrorCode != etcderrors.EcodeNodeExist {
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
				return nil, etcderrors.NewRequestError(etcderrors.EcodeNotFile, path)
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
			return nil, etcderrors.NewRequestError(etcderrors.EcodeKeyNotFound, path)
		}
	}

	return &etcd.Response{
		Action:    "get",
		Node:      current,
		EtcdIndex: c.etcdIndex,
	}, nil
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
