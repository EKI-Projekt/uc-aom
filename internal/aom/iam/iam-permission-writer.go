// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package iam

import (
	"fmt"
	"io"
	"text/template"

	log "github.com/sirupsen/logrus"
)

const iamPermissionTemplate = `
{
  "compatibility-version": "1.0",
  "service": "uc-auth",
  "group-display-name": "Web server - Protected applications",
  "permissions": [
    {
      "id": "{{.Id}}.access",
      "display-name": "Access to {{.AddOnTitle}}",
      "no-auth-option": {{.NoAuthOpt}}
    }
  ]
}
`

type IamPermission struct {
	AddOnTitle string
	Id         string
	NoAuthOpt  bool
}

type IamPermissionWriter struct {
	permissionTemplate *template.Template
	iamPermissionsPath string
	writeToFile        WriteToFileCmd
	deleteFile         DeleteFileCmd
}

// Given a filepath, an io.Writer instance will be created and passed
// to the function which is responsible for creating and saving the file's contents.
type WriteToFileCmd func(filepath string, factory func(io.Writer) error) error

// Deletes the file at the given filepath.
type DeleteFileCmd func(filepath string) error

func (w *IamPermissionWriter) getPermissionsFilepath(id string) string {
	return getFilepath(w.iamPermissionsPath, id)
}

func getFilepath(parent string, name string) string {
	return fmt.Sprintf("%s/%s-proxy.json", parent, name)
}

func NewIamPermissionWriter(
	iamPermissionsPath string,
	writeToFile WriteToFileCmd,
	deleteFile DeleteFileCmd) *IamPermissionWriter {

	tmpl, err := template.New("IamPermission").Parse(iamPermissionTemplate)

	if err != nil {
		panic(err)
	}

	return &IamPermissionWriter{
		permissionTemplate: tmpl,
		iamPermissionsPath: iamPermissionsPath,
		writeToFile:        writeToFile,
		deleteFile:         deleteFile}
}

func (w *IamPermissionWriter) Create(iamPermission *IamPermission) error {
	log.Tracef("IamPermissionWriter/Create")
	filepath := w.getPermissionsFilepath(iamPermission.Id)

	executeTemplate := func(writer io.Writer) error {
		err := w.permissionTemplate.Execute(writer, iamPermission)
		return err
	}

	err := w.writeToFile(filepath, executeTemplate)
	if err != nil {
		return err
	}

	return nil
}

func (w *IamPermissionWriter) Delete(id string) error {
	log.Tracef("IamPermissionWriter/Delete")
	path := w.getPermissionsFilepath(id)
	err := w.deleteFile(path)
	if err != nil {
		return err
	}

	return nil
}
