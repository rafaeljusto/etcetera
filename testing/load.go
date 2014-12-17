package main

import (
	"fmt"

	"github.com/rafaeljusto/etcetera"
)

// Config is an example of configuration file with the common types of fields
type Config struct {
	Key1 SubConfig1        `etcd:"/key1"`
	Key2 []SubConfig2      `etcd:"/key2"`
	Key3 map[string]string `etcd:"/key3"`
}

// SubConfig1 is an example configuration field with subfield with primitive types
type SubConfig1 struct {
	Subkey1 string `etcd:"/subkey1"`
	Subkey2 int    `etcd:"/subkey2"`
}

// SubConfig2 is another example of configuration field with subfield with primitive types
type SubConfig2 struct {
	Subkey1 int64 `etcd:"/subkey1"`
	Subkey2 bool  `etcd:"/subkey2"`
}

func main() {
	var config Config

	etc, err := etcetera.NewClient([]string{
		"http://127.0.0.1:4001",
		"http://127.0.0.1:4002",
		"http://127.0.0.1:4003",
	}, &config)

	if err != nil {
		fmt.Println(err)
		return
	}

	if err := etc.Load(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Loaded! %+v\n", config)
}