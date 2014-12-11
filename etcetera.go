// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/coreos/go-etcd/etcd"
)

var (
	// ErrInvalidConfig alert whenever you try to use something that is not a structure in the Save
	// function, or something that is not a pointer to a structure in the Load funciton
	ErrInvalidConfig = errors.New("etcetera: configuration must be a structure or a pointer to a structure")

	// ErrNotInitialized alert when you pass a structure to the Load function that has a map attribute
	// that is not initialized
	ErrNotInitialized = errors.New("etcetera: configuration has fields that are not initialized (map)")

	// ErrFieldNotMapped alert whenever you try to access a field that wasn't loaded in the client
	// structure. If we don't load the field before we cannot determinate the path or version
	ErrFieldNotMapped = errors.New("etcetera: trying to retrieve information of a field that wasn't previously loaded")
)

// https://github.com/coreos/etcd/blob/master/error/error.go
const (
	etcdErrorCodeKeyNotFound  etcdErrorCode = 100 // used in tests
	etcdErrorCodeNotFile      etcdErrorCode = 102 // used in tests
	etcdErrorCodeNodeExist    etcdErrorCode = 105
	etcdErrorCodeRaftInternal etcdErrorCode = 300 // used in tests
)

type etcdErrorCode int

// Client stores the etcd connection, the configuration instance that we are managing and some extra
// informations that are useful for controlling path versions and making the API simpler
type Client struct {
	etcdClient client
	config     reflect.Value

	// info creates a correlation between a path to a info structure that stores some extra
	// information and make the API usage easier
	info map[string]info
}

type info struct {
	field   reflect.Value
	version uint64
}

// NewClient internally build a etcd client object (go-etcd library). This internal object will not
// be visible to make the API simpler
func NewClient(machines []string, config interface{}) (*Client, error) {
	configValue := reflect.ValueOf(config)

	if configValue.Kind() != reflect.Ptr ||
		configValue.Elem().Kind() != reflect.Struct {

		return nil, ErrInvalidConfig
	}

	return &Client{
		etcdClient: etcd.NewClient(machines),
		config:     configValue,
		info:       make(map[string]info),
	}, nil
}

// Save stores a structure in etcd. Only attributes with the tag 'etcd' are going to be saved.
// Supported types are 'struct', 'slice', 'map', 'string', 'int', 'int64' and 'bool'
func (c *Client) Save() error {
	return c.save(c.config, "")
}

func (c *Client) save(config reflect.Value, pathSuffix string) error {
	if config.Kind() == reflect.Ptr {
		config = config.Elem()
	} else if config.Kind() != reflect.Struct {
		return ErrInvalidConfig
	}

	for i := 0; i < config.NumField(); i++ {
		fieldType := config.Type().Field(i)
		fieldValue := config.Field(i)

		path := pathSuffix + fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			if err := c.save(fieldValue, path); err != nil {
				return err
			}

		case reflect.Map:
			if _, err := c.etcdClient.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for _, key := range fieldValue.MapKeys() {
				value := fieldValue.MapIndex(key)

				if _, err := c.etcdClient.Set(path+"/"+key.String(), value.String(), 0); err != nil {
					return err
				}
			}

		case reflect.Slice:
			if _, err := c.etcdClient.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for i := 0; i < fieldValue.Len(); i++ {
				value := fieldValue.Index(i)

				if value.Kind() == reflect.Struct {
					tmpPath := fmt.Sprintf("%s/%d", path, i)

					if _, err := c.etcdClient.CreateDir(tmpPath, 0); err != nil && !alreadyExistsError(err) {
						return err
					}

					if err := c.save(value, tmpPath); err != nil {
						return err
					}

				} else {
					if _, err := c.etcdClient.CreateInOrder(path, value.String(), 0); err != nil {
						return err
					}
				}
			}

		case reflect.String:
			value := fieldValue.Interface().(string)
			if _, err := c.etcdClient.Set(path, value, 0); err != nil {
				return err
			}

		case reflect.Int:
			value := fieldValue.Interface().(int)
			if _, err := c.etcdClient.Set(path, strconv.FormatInt(int64(value), 10), 0); err != nil {
				return err
			}

		case reflect.Int64:
			value := fieldValue.Interface().(int64)
			if _, err := c.etcdClient.Set(path, strconv.FormatInt(value, 10), 0); err != nil {
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

			if _, err := c.etcdClient.Set(path, valueStr, 0); err != nil {
				return err
			}
		}

		c.info[path] = info{
			field: fieldValue,
		}
	}

	return nil
}

func alreadyExistsError(err error) bool {
	etcderr, ok := err.(*etcd.EtcdError)
	if !ok {
		return false
	}

	return etcderr.ErrorCode == int(etcdErrorCodeNodeExist)
}

// Load retrieves the data from the etcd into the given structure. Only attributes with the tag
// 'etcd' will be filled. Supported types are 'struct', 'slice', 'map', 'string', 'int', 'int64' and
// 'bool'
func (c *Client) Load() error {
	return c.load(c.config, "")
}

func (c *Client) load(config reflect.Value, pathSuffix string) error {
	if config.Kind() != reflect.Ptr {
		return ErrInvalidConfig
	}
	config = config.Elem()

	for i := 0; i < config.NumField(); i++ {
		fieldType := config.Type().Field(i)
		fieldValue := config.Field(i)

		path := pathSuffix + fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}

		if fieldValue.Kind() == reflect.Struct {
			if err := c.load(fieldValue.Addr(), path); err != nil {
				return err
			}

			c.info[path] = info{
				field: fieldValue,
			}

			continue
		}

		response, err := c.etcdClient.Get(path, true, true)
		if err != nil {
			return err
		}

		if err := c.fillValue(fieldValue, response); err != nil {
			return err
		}
	}

	return nil
}

// Watch keeps track of a specific field in etcd using a long polling strategy. When a change is
// detected the callback function will run. When you want to stop watching the field, just close the
// returning channel
func (c *Client) Watch(field interface{}, callback func()) (chan<- bool, error) {
	fieldValue := reflect.ValueOf(field)

	var path string
	var info info

	found := false
	for path, info = range c.info {
		if info.field.CanAddr() != fieldValue.CanAddr() {
			continue

		} else if info.field.CanAddr() {
			if info.field.Addr().Pointer() == fieldValue.Addr().Pointer() {
				found = true
				break
			}

		} else {
			// TODO?!?
		}
	}

	if !found {
		return nil, ErrFieldNotMapped
	}

	stop := make(chan bool)
	receiver := make(chan *etcd.Response)

	go c.etcdClient.Watch(path, info.version+1, true, receiver, stop)

	go func() {
		for {
			select {
			case response := <-receiver:
				if err := c.fillValue(fieldValue, response); err != nil {
					// TODO: Error setting the value
					continue
				}

				callback()

			case <-stop:
				return
			}
		}
	}()

	return nil, nil
}

func (c *Client) fillValue(fieldValue reflect.Value, response *etcd.Response) error {
	switch fieldValue.Kind() {
	case reflect.Map:
		if fieldValue.IsNil() {
			return ErrNotInitialized
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
			for _, node := range response.Node.Nodes {
				newElement := reflect.New(fieldValue.Type().Elem())
				if err := c.load(newElement, node.Key); err != nil {
					return err
				}

				fieldValue.Set(reflect.Append(fieldValue, newElement.Elem()))
			}

		case reflect.String:
			for _, node := range response.Node.Nodes {
				fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(node.Value)))
			}

		case reflect.Int:
			for _, node := range response.Node.Nodes {
				value, err := strconv.ParseInt(node.Value, 10, 64)
				if err != nil {
					return err
				}

				fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(int(value))))
			}

		case reflect.Int64:
			for _, node := range response.Node.Nodes {
				value, err := strconv.ParseInt(node.Value, 10, 64)
				if err != nil {
					return err
				}

				fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(value)))
			}

		case reflect.Bool:
			for _, node := range response.Node.Nodes {
				if node.Value == "true" {
					fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(true)))
				} else if node.Value == "false" {
					fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(false)))
				}
			}
		}

	case reflect.String:
		fieldValue.SetString(response.Node.Value)

	case reflect.Int, reflect.Int64:
		value, err := strconv.ParseInt(response.Node.Value, 10, 64)
		if err != nil {
			return err
		}

		fieldValue.SetInt(value)

	case reflect.Bool:
		if response.Node.Value == "true" {
			fieldValue.SetBool(true)
		} else if response.Node.Value == "false" {
			fieldValue.SetBool(false)
		}
	}

	c.info[response.Node.Key] = info{
		field:   fieldValue,
		version: response.Node.ModifiedIndex,
	}

	return nil
}
