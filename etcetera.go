// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

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

	// ErrFieldNotAddr is throw when a field that cannot be addressable is used in a place that we
	// need the pointer to identify the path related to the field
	ErrFieldNotAddr = errors.New("etcetera: field must be a pointer or an addressable value")
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

	c := &Client{
		etcdClient: etcd.NewClient(machines),
		config:     configValue,
		info:       make(map[string]info),
	}

	c.preload(c.config, "")
	return c, nil
}

func (c *Client) preload(field reflect.Value, pathSuffix string) {
	field = field.Elem()

	switch field.Kind() {
	case reflect.Struct:
		for i := 0; i < field.NumField(); i++ {
			subfield := field.Field(i)
			subfieldType := field.Type().Field(i)

			path := subfieldType.Tag.Get("etcd")
			if len(path) == 0 {
				continue
			}
			path = pathSuffix + path

			c.preload(subfield.Addr(), path)
		}
	}

	if len(pathSuffix) == 0 {
		pathSuffix = "/"
	}

	c.info[pathSuffix] = info{
		field: field,
	}
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
		field := config.Field(i)
		fieldType := config.Type().Field(i)

		path := fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}
		path = pathSuffix + path

		switch field.Kind() {
		case reflect.Struct:
			if err := c.save(field, path); err != nil {
				return err
			}

		case reflect.Map:
			if _, err := c.etcdClient.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for _, key := range field.MapKeys() {
				value := field.MapIndex(key)

				if _, err := c.etcdClient.Set(path+"/"+key.String(), value.String(), 0); err != nil {
					return err
				}
			}

		case reflect.Slice:
			if _, err := c.etcdClient.CreateDir(path, 0); err != nil && !alreadyExistsError(err) {
				return err
			}

			for i := 0; i < field.Len(); i++ {
				item := field.Index(i)

				if item.Kind() == reflect.Struct {
					tmpPath := fmt.Sprintf("%s/%d", path, i)

					if _, err := c.etcdClient.CreateDir(tmpPath, 0); err != nil && !alreadyExistsError(err) {
						return err
					}

					if err := c.save(item, tmpPath); err != nil {
						return err
					}

				} else {
					if _, err := c.etcdClient.CreateInOrder(path, item.String(), 0); err != nil {
						return err
					}
				}
			}

		case reflect.String:
			value := field.Interface().(string)
			if _, err := c.etcdClient.Set(path, value, 0); err != nil {
				return err
			}

		case reflect.Int:
			value := field.Interface().(int)
			if _, err := c.etcdClient.Set(path, strconv.FormatInt(int64(value), 10), 0); err != nil {
				return err
			}

		case reflect.Int64:
			value := field.Interface().(int64)
			if _, err := c.etcdClient.Set(path, strconv.FormatInt(value, 10), 0); err != nil {
				return err
			}

		case reflect.Bool:
			value := field.Interface().(bool)

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
			field: field,
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
		field := config.Field(i)
		fieldType := config.Type().Field(i)

		path := fieldType.Tag.Get("etcd")
		if len(path) == 0 {
			continue
		}
		path = pathSuffix + path

		response, err := c.etcdClient.Get(path, true, true)
		if err != nil {
			return err
		}

		if err := c.fillField(field, response.Node, path); err != nil {
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
	if fieldValue.Kind() == reflect.Ptr {
		fieldValue = fieldValue.Elem()

	} else if !fieldValue.CanAddr() {
		return nil, ErrFieldNotAddr
	}

	var path string
	var info info

	found := false
	for path, info = range c.info {
		// Match the pointer, type and name to avoid problems for struct and first field that have the
		// same memory address
		if info.field.Addr().Pointer() == fieldValue.Addr().Pointer() &&
			info.field.Type().Name() == fieldValue.Type().Name() &&
			info.field.Kind() == fieldValue.Kind() {

			found = true
			break
		}
	}

	if !found {
		return nil, ErrFieldNotMapped
	}

	stop := make(chan bool)
	receiver := make(chan *etcd.Response)

	// We are always retrieving the last version (index) of the path
	go c.etcdClient.Watch(path, 0, true, receiver, stop)

	go func() {
		for {
			select {
			case response := <-receiver:
				if response != nil {
					// When watching a directory (slice, map or structure) the response will be from the node
					// that changed and not the entire directory. So we need to query the directory again with
					// recursion to load it correctly.
					response, err := c.etcdClient.Get(path, true, true)
					if err == nil {
						c.fillField(fieldValue, response.Node, path)
						callback()
					}
				}

			case <-stop:
				return
			}
		}
	}()

	return stop, nil
}

func (c *Client) fillField(field reflect.Value, node *etcd.Node, pathSuffix string) error {
	switch field.Kind() {
	case reflect.Struct:
		for i := 0; i < field.NumField(); i++ {
			subfield := field.Field(i)
			subfieldType := field.Type().Field(i)

			path := subfieldType.Tag.Get("etcd")
			if len(path) == 0 {
				continue
			}
			path = pathSuffix + path

			for _, child := range node.Nodes {
				if path == child.Key {
					if err := c.fillField(subfield, child, path); err != nil {
						return err
					}
					break
				}
			}
		}

	case reflect.Map:
		field.Set(reflect.MakeMap(field.Type()))

		for _, node := range node.Nodes {
			pathParts := strings.Split(node.Key, "/")

			field.SetMapIndex(
				reflect.ValueOf(pathParts[len(pathParts)-1]),
				reflect.ValueOf(node.Value),
			)
		}

	case reflect.Slice:
		field.Set(reflect.MakeSlice(field.Type(), 0, len(node.Nodes)))

		switch field.Type().Elem().Kind() {
		case reflect.Struct:
			for i, item := range node.Nodes {
				newStruct := reflect.New(field.Type().Elem()).Elem()

			SubitemLoop:
				for _, subitem := range item.Nodes {
					for j := 0; j < newStruct.NumField(); j++ {
						subfield := newStruct.Field(j)
						subfieldType := newStruct.Type().Field(j)

						path := subfieldType.Tag.Get("etcd")
						if len(path) == 0 {
							continue
						}
						path = fmt.Sprintf("%s/%d%s", pathSuffix, i, path)

						if path == subitem.Key {
							if err := c.fillField(subfield, subitem, path); err != nil {
								return err
							}
							continue SubitemLoop
						}
					}
				}
				field.Set(reflect.Append(field, newStruct))
			}

		case reflect.String:
			for _, node := range node.Nodes {
				field.Set(reflect.Append(field, reflect.ValueOf(node.Value)))
			}

		case reflect.Int:
			for _, node := range node.Nodes {
				value, err := strconv.ParseInt(node.Value, 10, 64)
				if err != nil {
					return err
				}

				field.Set(reflect.Append(field, reflect.ValueOf(int(value))))
			}

		case reflect.Int64:
			for _, node := range node.Nodes {
				value, err := strconv.ParseInt(node.Value, 10, 64)
				if err != nil {
					return err
				}

				field.Set(reflect.Append(field, reflect.ValueOf(value)))
			}

		case reflect.Bool:
			for _, node := range node.Nodes {
				if node.Value == "true" {
					field.Set(reflect.Append(field, reflect.ValueOf(true)))
				} else if node.Value == "false" {
					field.Set(reflect.Append(field, reflect.ValueOf(false)))
				}
			}
		}

	case reflect.String:
		field.SetString(node.Value)

	case reflect.Int, reflect.Int64:
		value, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil {
			return err
		}

		field.SetInt(value)

	case reflect.Bool:
		if node.Value == "true" {
			field.SetBool(true)
		} else if node.Value == "false" {
			field.SetBool(false)
		}
	}

	c.info[node.Key] = info{
		field:   field,
		version: node.ModifiedIndex,
	}

	return nil
}

// Version returns the current version of a field retrieved from etcd. It does not query etcd for
// the latest version. When the field was not retrieved from etcd yet, the version 0 is returned
func (c *Client) Version(field interface{}) (uint64, error) {
	fieldValue := reflect.ValueOf(field)
	if fieldValue.Kind() == reflect.Ptr {
		fieldValue = fieldValue.Elem()

	} else if !fieldValue.CanAddr() {
		return 0, ErrFieldNotAddr
	}

	for _, info := range c.info {
		// Match the pointer, type and name to avoid problems for struct and first field that have the
		// same memory address
		if info.field.Addr().Pointer() == fieldValue.Addr().Pointer() &&
			info.field.Type().Name() == fieldValue.Type().Name() &&
			info.field.Kind() == fieldValue.Kind() {

			return info.version, nil
		}
	}

	return 0, ErrFieldNotMapped
}
