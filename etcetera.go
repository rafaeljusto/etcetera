// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"fmt"
	"reflect"
	"strconv"

	etcderrors "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

// Save stores a structure in etcd. Only attributes with the tag 'etcd' are going to be saved.
// Supported types are 'struct', 'slice', 'map', 'string', 'int', 'int64' and 'bool'
func Save(config interface{}, client *etcd.Client) error {
	return save(config, client, "")
}

func save(config interface{}, client *etcd.Client, pathSuffix string) error {
	st := reflect.ValueOf(config).Elem()

	for i := 0; i < st.NumField(); i++ {
		fieldType := st.Type().Field(i)
		fieldValue := st.Field(i)

		path := pathSuffix + fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			if err := save(fieldValue.Addr().Interface(), client, path); err != nil {
				return err
			}

		case reflect.Map:
			if _, err := client.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for _, key := range fieldValue.MapKeys() {
				value := fieldValue.MapIndex(key)

				if _, err := client.Set(path+"/"+key.String(), value.String(), 0); err != nil {
					return err
				}
			}

		case reflect.Slice:
			if _, err := client.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for i := 0; i < fieldValue.Len(); i++ {
				value := fieldValue.Index(i)

				if value.Kind() == reflect.Struct {
					tmpPath := fmt.Sprintf("%s/%d", path, i)

					if _, err := client.CreateDir(tmpPath, 0); err != nil && !alreadyExistsError(err) {
						return err
					}

					if err := save(value.Addr().Interface(), client, tmpPath); err != nil {
						return err
					}

				} else {
					if _, err := client.CreateInOrder(path, value.String(), 0); err != nil {
						return err
					}
				}
			}

		case reflect.String:
			value := fieldValue.Interface().(string)
			if _, err := client.Set(path, value, 0); err != nil {
				return err
			}

		case reflect.Int:
			value := fieldValue.Interface().(int)
			if _, err := client.Set(path, strconv.FormatInt(int64(value), 10), 0); err != nil {
				return err
			}

		case reflect.Int64:
			value := fieldValue.Interface().(int64)
			if _, err := client.Set(path, strconv.FormatInt(value, 10), 0); err != nil {
				return err
			}

		case reflect.Bool:
			value := fieldValue.Interface().(bool)

			var valueStr string
			if value {
				valueStr = "true"
			} else {
				valueStr = "false"
			}

			if _, err := client.Set(path, valueStr, 0); err != nil {
				return err
			}
		}
	}

	return nil
}

// Load retrieves the data from the etcd into the given structure. Only attributes with the tag
// 'etcd' will be filled. Supported types are 'struct', 'slice', 'map', 'string', 'int', 'int64' and
// 'bool'
func Load(config interface{}, client *etcd.Client) error {
	return load(config, client, "")
}

func load(config interface{}, client *etcd.Client, pathSuffix string) error {
	st := reflect.ValueOf(config).Elem()

	for i := 0; i < st.NumField(); i++ {
		fieldType := st.Type().Field(i)
		fieldValue := st.Field(i)

		path := pathSuffix + fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			if err := load(fieldValue.Addr().Interface(), client, path); err != nil {
				return err
			}

		case reflect.Map:
			if fieldValue.IsNil() {
				return fmt.Errorf("Map must be initialized")
			}

			response, err := client.Get(path, true, true)
			if err != nil {
				return err
			}

			for _, node := range response.Node.Nodes {
				fieldValue.SetMapIndex(
					reflect.ValueOf(node.Key),
					reflect.ValueOf(node.Value),
				)
			}

		case reflect.Slice:
			switch fieldValue.Type().Elem().Kind() {
			case reflect.Struct:
				response, err := client.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					newElement := reflect.New(fieldValue.Type().Elem())
					if err := load(newElement.Interface(), client, node.Key); err != nil {
						return err
					}

					fieldValue.Set(reflect.Append(fieldValue, newElement.Elem()))
				}

			case reflect.String:
				response, err := client.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(node.Value)))
				}

			case reflect.Int, reflect.Int64:
				response, err := client.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					value, err := strconv.ParseInt(node.Value, 10, 64)
					if err != nil {
						return err
					}

					fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(value)))
				}

			case reflect.Bool:
				response, err := client.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					if node.Value == "true" {
						fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(true)))
					} else if node.Value == "false" {
						fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(false)))
					}
				}
			}

		case reflect.String:
			response, err := client.Get(path, false, false)
			if err != nil {
				return err
			}

			fieldValue.SetString(response.Node.Value)

		case reflect.Int, reflect.Int64:
			response, err := client.Get(path, false, false)
			if err != nil {
				return err
			}

			value, err := strconv.ParseInt(response.Node.Value, 10, 64)
			if err != nil {
				return err
			}

			fieldValue.SetInt(value)

		case reflect.Bool:
			response, err := client.Get(path, false, false)
			if err != nil {
				return err
			}

			if response.Node.Value == "true" {
				fieldValue.SetBool(true)
			} else if response.Node.Value == "false" {
				fieldValue.SetBool(false)
			}
		}
	}

	return nil
}

func alreadyExistsError(err error) bool {
	etcderr, ok := err.(*etcd.EtcdError)
	if !ok {
		return false
	}

	return etcderr.ErrorCode == etcderrors.EcodeNodeExist
}
