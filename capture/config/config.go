// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// config.go [created: Sat, 25 May 2013]

/*
This package defines a configuration file format meant to house many client ids.
The idea was to have a new CLI for capture. But one does not exist.
*/
package config

import (
	"github.com/bmatsuo1/go-janrain/capture"

	"encoding/json"
	"io"
	"os"
)

type Config struct {
	Cli     *CLIConfig                            `json:"cli,omitempty"`
	Apps    map[string]*AppConfig                 `json:"apps"`
	Clients map[string]*capture.ClientCredentials `json:"clients,omitempty"`
}

type AppConfig struct {
	Domain        string                                `json:"domain"`
	AppId         string                                `json:"app_id,omitempty"`
	DefaultClient string                                `json:"default_client,omitempty"`
	Clients       map[string]*capture.ClientCredentials `json:"clients,omitempty"`
}

type CLIConfig struct {
}

type indentedJSON struct {
	val    interface{}
	prefix string
	indent string
}

func newIndentedJSON(val interface{}) *indentedJSON {
	return &indentedJSON{
		val:    val,
		indent: "\t",
	}
}

func (js *indentedJSON) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(js.val, js.prefix, js.indent)
}

func WriteJSON(w io.Writer, config *Config) error {
	enc := json.NewEncoder(w)
	return enc.Encode(newIndentedJSON(config))
}

func WriteFileJSON(filename string, config *Config) error {
	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer handle.Close()
	return WriteJSON(handle, config)
}

func ReadJSON(r io.Reader) (*Config, error) {
	config := new(Config)
	dec := json.NewDecoder(r)
	err := dec.Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func ReadFileJSON(filename string) (*Config, error) {
	handle, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	return ReadJSON(handle)
}
