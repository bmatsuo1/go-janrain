// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// config.go [created: Sat, 25 May 2013]

package config

import (
	"github.com/bmatsuo1/go-janrain/capture"

	"encoding/json"
	"io"
	"os"
)

type Config struct {
	Cli     *CLIConfig                            `json:"cli" yaml:"cli,omitempty"`
	Apps    map[string]*AppConfig                 `json:"apps" yaml:"apps"`
	Clients map[string]*capture.ClientCredentials `json:"clients,omitempty" yaml:"clients,omitempty"`
}

type AppConfig struct {
	Domain        string                                `json:"domain" yaml:"domain"`
	AppId         string                                `json:"app_id,omitempty" yaml:"app_id,omitempty"`
	DefaultClient string                                `json:"default_client,omitempty" yaml:"default_client,omitempty"`
	Clients       map[string]*capture.ClientCredentials `json:"clients,omitempty" yaml:"clients,omitempty"`
}

type CLIConfig struct {
}

// json

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

func WriteConfigJSON(w io.Writer, config *Config) error {
	enc := json.NewEncoder(w)
	return enc.Encode(newIndentedJSON(config))
}

func ReadConfigJSON(r io.Reader) (*Config, error) {
	config := new(Config)
	dec := json.NewDecoder(r)
	err := dec.Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func ReadConfigFileJSON(filename string) (*Config, error) {
	handle, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	return ReadConfigJSON(handle)
}
