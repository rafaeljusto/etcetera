// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package etcetera is etcd client that uses a tagged struct to save and load values
//
// Behavior
//
// We took some decisions when creating this library taking into account that less is more. The
// decisions are all listed bellow.
//
// Always retrieving the last index: For the use case that we thought, there's no reason (for now)
// to retrieve an intermediate state of a field. We are always looking for the current value in
// etcd. But we store the index of all attributes retrieved from etcd so that the user wants to know
// it (in the library we use "version" instead of "index" because it appears to have a better
// context).
//
// Setting unlimited TTL: The type of data that we store in etcd (configuration values) don't need a
// TTL. Or at least we did not imagine any case when it does need a TTL.
//
// Ignoring errors occurred in watch: When something goes wrong while retrieving or parsing the data
// from etcd, we prefer to silent drop the update instead of setting a strange value to the
// configuration field. Another problem is to create a good API to notify about errors occurred in
// watch, the first idea is to use a channel for errors, but it doesn't appears to be a elegant
// approach, and more than that, what the user can do with this error? Well, we are still thinking
// about it.
//
// Ignoring "directory already exist" errors: If the directory already exists, great! We go on and
// create the structure under this directory. There's no reason to stop everything because of this
// error.
//
// Not allowing URI in structure's field tag: This was a change made on 2014-12-17. I thought that
// leaving the decision to the user to create an URI in a structure's field tag could cause strange
// behaviors when structuring the configuration in etcd. So all the slashes in the field's tag will
// be replaced by hyphens, except if the slash is the first or last character
//
// Improve
//
// There are some issues that we still need to improve in the project. First, the code readability
// is terrible with all the reflection used, and with a good re-factory the repeated code could be
// reused. The full test coverage will ensure that the re-factory does not break anything.
//
// Second, when watching a field, you will receive a channel to notify when you want to stop
// watching. Now if you send a boolean false into the channel instead of closing it, we could have a
// strange behavior since there are two go routines listening on this channel (go-etcd and etcetera
// watch functions).
//
// And finally, we could have concurrency issues while updating configuration fields caused by the
// watch service. We still need to test the possible cases, but adding a read/write lock don't
// appears to be an elegant solution.
package etcetera
