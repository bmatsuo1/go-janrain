// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// client.go [created: Fri,  7 Jun 2013]

package capture

import (
	"github.com/bitly/go-simplejson"

	"encoding/json"
	"net/http"
	"net/url"
)

// an API client.
type Client struct {
	baseurl string
	auth    Authorization
	header  http.Header
	params  Params
	http    *http.Client
}

// construct a new API client. though auth can be nil it is generally
// recommended a value be passed as calls to the API generally require
// authorization.
func NewClient(baseurl string, auth Authorization) *Client {
	client := &Client{
		baseurl: baseurl,
		auth:    auth,
		params:  make(Params),
		header:  make(http.Header),
		http:    new(http.Client),
	}
	return client
}

// execute an API call with the Authorization used to initialize the client.
func (client *Client) Execute(method string, params Params) (*simplejson.Json, error) {
	return client.ExecuteAuth(nil, method, params)
}

// execute an API call with an Authorization that overrides the value used to
// initialize the client.
func (client *Client) ExecuteAuth(auth Authorization, method string, params Params) (*simplejson.Json, error) {
	uri, header, values, err := client.merge(method, params)
	if err != nil {
		return nil, err
	}

	if auth == nil {
		auth = client.auth
	}
	//fmt.Println("auth ", auth)
	if auth != nil {
		err = auth.Authorize(uri, header, values)
		if err != nil {
			return nil, err
		}
	}

	req, err := prepare("POST", uri, header, values)
	return client.perform(req)
}

func (client *Client) merge(method string, params Params) (*url.URL, http.Header, url.Values, error) {
	endpoint := client.baseurl
	if method[0] != '/' {
		endpoint += "/"
	}
	endpoint += method
	uri, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, nil, err
	}

	header := make(http.Header)
	for k, v := range client.header {
		header[k] = v
	}

	mergedparams := make(Params, len(client.params)+len(params))
	for k, v := range client.params {
		mergedparams[k] = v
	}
	for k, v := range params {
		mergedparams[k] = v
	}
	values, err := mergedparams.formValues()
	if err != nil {
		return nil, nil, nil, err
	}

	return uri, header, values, nil
}

func (client *Client) perform(req *http.Request) (*simplejson.Json, error) {
	resp, err := client.http.Do(req)
	if err != nil {
		return nil, HttpTransportError{err}
	}
	defer resp.Body.Close()
	switch mime := resp.Header.Get("Content-Type"); mime {
	case "application/json", "text/json":
		dec := json.NewDecoder(resp.Body)
		js := new(simplejson.Json)
		err := dec.Decode(js)
		if err != nil {
			return nil, JsonDecoderError{err}
		}
		if js.Get("stat").MustString() != "ok" {
			return nil, NewRemoteError(js)
		}
		return js, nil
	default:
		return nil, ContentTypeError(mime)
	}
}
