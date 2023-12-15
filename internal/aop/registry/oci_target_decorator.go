// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

// Decorates a oras.Target with the add-on's version.
type AddOnTarget interface {
	oras.Target
	AddOnVersion() string
}

type AddOnRepositoryTarget interface {
	AddOnTarget
	AddOnRepository() string
}

// Decorates a oras.Target with a tag.
type TargetDecorator struct {
	decorate oras.Target
	tag      string
}

// Returns the add-on's version.
func (r TargetDecorator) AddOnVersion() string {
	return r.tag
}

func (r TargetDecorator) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	return r.decorate.Exists(ctx, target)
}

func (r TargetDecorator) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	return r.decorate.Fetch(ctx, target)
}

func (r TargetDecorator) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	return r.decorate.Push(ctx, expected, content)
}

func (r TargetDecorator) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	return r.decorate.Resolve(ctx, reference)
}

func (r TargetDecorator) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	return r.decorate.Tag(ctx, desc, reference)
}

// Returns a new instance that decorates decorate with the given tag.
func NewOciTargetDecorator(decorate oras.Target, tag string) TargetDecorator {
	return TargetDecorator{decorate: decorate, tag: tag}
}

type RepositoryTargetDecorator struct {
	TargetDecorator
	repository string
}

// Returns a new instance that decorates decorate with the given tag.
func NewOciRepositoryTargetDecorator(decorate oras.Target, tag string, repository string) *RepositoryTargetDecorator {
	target := NewOciTargetDecorator(decorate, tag)
	return &RepositoryTargetDecorator{TargetDecorator: target, repository: repository}
}

func (t *RepositoryTargetDecorator) AddOnRepository() string {
	return t.repository
}
