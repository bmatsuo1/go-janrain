// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// client.go [created: Fri,  7 Jun 2013]

package capture

import (
	"github.com/bitly/go-simplejson"

	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func prepare(method string, uri *url.URL, header http.Header, values url.Values) (*http.Request, error) {
	var body io.Reader
	if method == "POST" {
		body = strings.NewReader(values.Encode())
		header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	return req, nil
}

// API request parameters.
type Params map[string]interface{}

func (ps Params) formValues() (url.Values, error) {
	vals := make(url.Values, len(ps))
	for k, v := range ps {
		val, ok := v.(string)
		if !ok {
			p, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			val = string(p)
		}
		vals.Set(k, val)
	}
	return vals, nil
}

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
func (client *Client) Execute(method string, header http.Header, params Params) (*simplejson.Json, error) {
	return client.ExecuteAuth(nil, method, header, params)
}

// define a parameter that should be included in every request.
func (client *Client) Default(param string, value interface{}) {
	client.params[param] = value
}

// remove a default value previously specified with client.Default.
func (client *Client) DefaultClear(param string) {
	delete(client.params, param)
}

// execute an API call with an Authorization that overrides the value used to
// initialize the client.
func (client *Client) ExecuteAuth(auth Authorization, method string, header http.Header, params Params) (*simplejson.Json, error) {
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