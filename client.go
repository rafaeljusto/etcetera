// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package etcetera

import (
	"github.com/coreos/go-etcd/etcd"
)

type client interface {
	CreateDir(string, uint64) (*etcd.Response, error)
	CreateInOrder(string, string, uint64) (*etcd.Response, error)
	Set(string, string, uint64) (*etcd.Response, error)
	Get(string, bool, bool) (*etcd.Response, error)
}
