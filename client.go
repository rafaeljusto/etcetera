// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"github.com/coreos/go-etcd/etcd"
)

type client interface {
	CreateDir(path string, ttl uint64) (*etcd.Response, error)
	CreateInOrder(path, value string, ttl uint64) (*etcd.Response, error)
	Set(path, value string, ttl uint64) (*etcd.Response, error)
	Get(path string, sort, recursive bool) (*etcd.Response, error)
	Watch(path string, waitIndex uint64, recursive bool, receiver chan *etcd.Response, stop chan bool) (*etcd.Response, error)
}
