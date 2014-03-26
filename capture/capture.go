// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// capture.go created Tue, 21 May 2013

/*
A lightly wrapped http client library for accessing the Janrain Capture API.

Client credentials

API requests can be authorized with client credentials, adding an HMAC-SHA1
signature to API requests.

	creds := capture.ClientCredentials{"myclientid", "myclientsecret"}
	client := capture.NewClient("https://myapp.janraincapture.com", &creds)
	resp, _ := client.Execute("/entity.count", nil, capture.Params{
		"type_name": "user",
	})
	fmt.Println(resp.Get("total_count").MustInt())

Access tokens

API requests targeting a single user can be authorized with an access token tied
to that user.

	token := capture.AccessToken(req.FormValue("access_token"))
	client := capture.NewClient("https://myapp.janraincapture.com", token)
	resp, _ := client.Execute("/entity", nil, nil)
	entity := resp.Get("result")

	// ...

Authorization override

The two authorization methods can be mixed within the same client using the
ExecuteAuth method.

	creds := capture.ClientCredentials{"myclientid", "myclientsecret"}
	client := capture.NewClient("https://myapp.janraincapture.com", &creds)

	// request authorized by access token
	token := capture.AccessToken(req.FormValue("access_token"))
	resp, _ := client.ExecuteAuth(token, "/entity", nil, nil)
	fmt.Println(resp)

	// request authorized by client credentials
	givenName := entity.Get("givenName").MustString()
	resp, _ = client.Execute("/entity.count", nil, capture.Params{
		"type_name": "user",
	})
	fmt.Println(resp)

	//..

Filter strings

Type safe filter strings can be generated using the package

	github.com/bmatsuo1/go-janrain/capture/filter.

See it's documentation for more information.

*/
package capture

import (
	"github.com/bitly/go-simplejson"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// the date and time formats used by Capture.
var (
	DateFormat = "2006-01-02"
	TimeFormat = "2006-01-02 15:04:05.999999999 -0700"
)

// create a time.Time from a datestamp retured by the Capture API
func Date(datestamp string) (time.Time, error) {
	return time.Parse(DateFormat, datestamp)
}

// create a datestamp for passing to Capture.
func Datestamp(t time.Time) string {
	return t.Format(DateFormat)
}

// create a time.Time from a timestamp returned by Capture.
func Time(timestamp string) (time.Time, error) {
	return time.Parse(TimeFormat, timestamp)
}

// create a timestamp for passing to Capture.
func Timestamp(t time.Time) string {
	return t.Format(TimeFormat)
}

// an error returned by Capture.
type RemoteError struct {
	RequestId   string
	Code        int
	Kind        string
	Description string
	Response    *simplejson.Json
	*HttpResponseData
}

// construct an error from a Capture response.
func NewRemoteError(resp *http.Response, js *simplejson.Json) RemoteError {
	return RemoteError{
		RequestId:   js.Get("request_id").MustString(),
		Code:        js.Get("code").MustInt(),
		Kind:        js.Get("error").MustString(),
		Description: js.Get("error_description").MustString(),
		Response:    js,
		HttpResponseData: &HttpResponseData{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       []byte(jsonStringer(js).String()), // totally gnarly. this is not a good thing
		},
	}
}

func (err RemoteError) HttpResponse() *HttpResponseData {
	return err.HttpResponseData
}

func (err RemoteError) Error() string {
	return fmt.Sprintf("[%s] %s", err.Kind, err.Description)
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
	r   *HttpResponseData
	Err error
}

func (err JsonDecoderError) Error() string {
	return err.Err.Error()
}

func (err JsonDecoderError) HttpResponse() *HttpResponseData {
	return err.r
}

// an unexpected content type returned by the API.
type ContentTypeError struct {
	*HttpResponseData
}

func (err *ContentTypeError) Error() string {
	return fmt.Sprintf("unexpected content-type %q", err.Header.Get("Content-Type"))
}

func (err *ContentTypeError) HttpResponse() *HttpResponseData {
	return err.HttpResponseData
}

type HttpResponse interface {
	HttpResponse() *HttpResponseData
}

type HttpResponseData struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (r *HttpResponseData) String() string {
	return fmt.Sprintf("[%d] %q", r.StatusCode, r.Body)
}

func ReadResponse(resp *http.Response) (*HttpResponseData, error) {
	p, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	r := &HttpResponseData{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       p,
	}
	return r, err
}

type jstringer struct {
	v interface{}
}

func (js jstringer) String() string {
	p, _ := json.Marshal(js.v)
	return string(p)
}

// a type that stringifies as json.
func jsonStringer(v interface{}) fmt.Stringer {
	return jstringer{v}
}
