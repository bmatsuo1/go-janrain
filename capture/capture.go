// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// capture.go created Tue, 21 May 2013

/*
A lightly wrapped http client library for accessing the Janrain Capture API.

Client credentials

API requests can be authorized using client credentials. This adds an HMAC-SHA1
signature to API requests.

	creds := capture.ClientCredentials{"myclientid", "myclientsecret"}
	client := capture.NewClient("https://myapp.janraincapture.com", creds)
	filter := fmt.Sprintf("created > '%v'", time.Now().Add(-5*time.Hour))
	resp, _ := client.Execute("/entity.find", capture.Params{
		"filter": filter
	})
	for _, entity := resp.Get("results").MustArray() {
		// ...
	}

Access tokens

API requests targeting a single user can also be authorized with an access token
tied to that user.

	token := capture.AccessToken(req.FormValue("access_token"))
	client := capture.NewClient("https://myapp.janraincapture.com", token)
	resp, _ := client.Execute("/entity")
	entity := resp.Get("result")
	// ...

Authorization override

The two authorization methods can be mixed within the same client using the
EntityAuth method.

	creds := capture.ClientCredentials{"myclientid", "myclientsecret"}
	client := capture.NewClient("https://myapp.janraincapture.com", creds)

	token := capture.AccessToken(req.FormValue("access_token"))
	resp, _ := client.ExecuteAuth("/entity", token)

	givenName := entity.Get("givenName").MustString()
	filter := capture.Filter("givenName =", givenName)
	resp, _ := client.Execute("/entity.find", capture.Params{
		"filter": filter,
	})
	//..
*/
package capture

import (
	"github.com/bitly/go-simplejson"

	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var DateFormat = "2006-01-02"
var TimeFormat = "2006-01-02 15:04:05.999999999 -0700"

func Time(timestamp string) (time.Time, error) {
	return time.Parse(TimeFormat, timestamp)
}

func Timestamp(t time.Time) string {
	return t.Format(TimeFormat)
}

func Datestamp(t time.Time) string {
	return t.Format(DateFormat)
}

func Date(datestamp string) (time.Time, error) {
	return time.Parse(DateFormat, datestamp)
}

// API request parameters
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

// an error returned by the Janrain Capture API.
type RemoteError struct {
	Code        int
	Kind        string
	Description string
	Response    *simplejson.Json
}

// construct an error from an API response.
func NewRemoteError(js *simplejson.Json) RemoteError {
	return RemoteError{
		Code:        js.Get("code").MustInt(),
		Kind:        js.Get("error").MustString(),
		Description: js.Get("error_description").MustString(),
		Response:    js,
	}
}

func (err RemoteError) Error() string {
	return fmt.Sprintf("%s (%d) %s", err.Kind, err.Code, err.Description)
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

// an error in http communication.
type HttpTransportError struct {
	Err error
}

func (err HttpTransportError) Error() string {
	return err.Err.Error()
}

// an error decoding a JSON response from the API.
type JsonDecoderError struct {
	Err error
}

func (err JsonDecoderError) Error() string {
	return err.Err.Error()
}

// an unexpected content type returned by the API.
type ContentTypeError string

func (err ContentTypeError) Error() string {
	return fmt.Sprintf("unexpected content-type %q", err)
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

func prepare(method string, uri *url.URL, header http.Header, values url.Values) (*http.Request, error) {
	//fmt.Println(uri)
	//fmt.Println(header)
	//fmt.Println(values.Encode())
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
	//fmt.Println(req)
	return req, nil
}

type jsonStringer struct {
	val interface{}
}

func (js jsonStringer) String() string {
	p, _ := json.Marshal(js)
	return string(p)
}

type jsonMap map[string]interface{}

func (m jsonMap) String() string {
	return jsonStringer{m}.String()
}
