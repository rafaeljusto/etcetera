// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	etcderrors "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

var (
	// ErrInvalidConfig alert whenever you try to use something that is not a structure in the Save
	// function, or something that is not a pointer to a structure in the Load funciton
	ErrInvalidConfig = errors.New("etcetera: configuration must be a structure or a pointer to a structure")

	// ErrNotInitialized alert when you pass a structure to the Load function that has a map attribute
	// that is not initialized
	ErrNotInitialized = errors.New("etcetera: configuration has fields that are not initialized (map)")
)

// Save stores a structure in etcd. Only attributes with the tag 'etcd' are going to be saved.
// Supported types are 'struct', 'slice', 'map', 'string', 'int', 'int64' and 'bool'
func Save(config interface{}, c client) error {
	return save(config, c, "")
}

func save(config interface{}, c client, pathSuffix string) error {
	st := reflect.ValueOf(config)
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	} else if st.Kind() != reflect.Struct {
		return ErrInvalidConfig
	}

	for i := 0; i < st.NumField(); i++ {
		fieldType := st.Type().Field(i)
		fieldValue := st.Field(i)

		path := pathSuffix + fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			if err := save(fieldValue.Interface(), c, path); err != nil {
				return err
			}

		case reflect.Map:
			if _, err := c.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for _, key := range fieldValue.MapKeys() {
				value := fieldValue.MapIndex(key)

				if _, err := c.Set(path+"/"+key.String(), value.String(), 0); err != nil {
					return err
				}
			}

		case reflect.Slice:
			if _, err := c.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for i := 0; i < fieldValue.Len(); i++ {
				value := fieldValue.Index(i)

				if value.Kind() == reflect.Struct {
					tmpPath := fmt.Sprintf("%s/%d", path, i)

					if _, err := c.CreateDir(tmpPath, 0); err != nil && !alreadyExistsError(err) {
						return err
					}

					if err := save(value.Interface(), c, tmpPath); err != nil {
						return err
					}

				} else {
					if _, err := c.CreateInOrder(path, value.String(), 0); err != nil {
						return err
					}
				}
			}

		case reflect.String:
			value := fieldValue.Interface().(string)
			if _, err := c.Set(path, value, 0); err != nil {
				return err
			}

		case reflect.Int:
			value := fieldValue.Interface().(int)
			if _, err := c.Set(path, strconv.FormatInt(int64(value), 10), 0); err != nil {
				return err
			}

		case reflect.Int64:
			value := fieldValue.Interface().(int64)
			if _, err := c.Set(path, strconv.FormatInt(value, 10), 0); err != nil {
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

			if _, err := c.Set(path, valueStr, 0); err != nil {
				return err
			}
		}
	}

	return nil
}

// Load retrieves the data from the etcd into the given structure. Only attributes with the tag
// 'etcd' will be filled. Supported types are 'struct', 'slice', 'map', 'string', 'int', 'int64' and
// 'bool'
func Load(config interface{}, c client) error {
	return load(config, c, "")
}

func load(config interface{}, c client, pathSuffix string) error {
	st := reflect.ValueOf(config)
	if st.Kind() != reflect.Ptr {
		return ErrInvalidConfig
	}
	st = st.Elem()

	for i := 0; i < st.NumField(); i++ {
		fieldType := st.Type().Field(i)
		fieldValue := st.Field(i)

		path := pathSuffix + fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			if err := load(fieldValue.Addr().Interface(), c, path); err != nil {
				return err
			}

		case reflect.Map:
			if fieldValue.IsNil() {
				return ErrNotInitialized
			}

			response, err := c.Get(path, true, true)
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
				response, err := c.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					newElement := reflect.New(fieldValue.Type().Elem())
					if err := load(newElement.Interface(), c, node.Key); err != nil {
						return err
					}

					fieldValue.Set(reflect.Append(fieldValue, newElement.Elem()))
				}

			case reflect.String:
				response, err := c.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(node.Value)))
				}

			case reflect.Int:
				response, err := c.Get(path, true, true)
				if err != nil {
					return err
				}

				for _, node := range response.Node.Nodes {
					value, err := strconv.ParseInt(node.Value, 10, 64)
					if err != nil {
						return err
					}

					fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(int(value))))
				}

			case reflect.Int64:
				response, err := c.Get(path, true, true)
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
				response, err := c.Get(path, true, true)
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
			response, err := c.Get(path, false, false)
			if err != nil {
				return err
			}

			fieldValue.SetString(response.Node.Value)

		case reflect.Int, reflect.Int64:
			response, err := c.Get(path, false, false)
			if err != nil {
				return err
			}

			value, err := strconv.ParseInt(response.Node.Value, 10, 64)
			if err != nil {
				return err
			}

			fieldValue.SetInt(value)

		case reflect.Bool:
			response, err := c.Get(path, false, false)
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
