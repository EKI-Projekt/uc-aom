// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package oraswrapper

import (
	"bytes"
	"context"

	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/content"
)

func PushAll(ctx context.Context, pusher content.Pusher, tuples ...*DescriptorBlobTuple) error {
	log.Tracef("PushAll")
	for _, tuple := range tuples {
		log.Debugf("Push to storage: %v", tuple.Desc)
		err := pusher.Push(ctx, *tuple.Desc, bytes.NewReader(tuple.Blob))
		if err != nil {
			return err
		}
	}
	return nil
}
