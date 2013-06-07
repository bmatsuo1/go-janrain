// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// auth.go created Wed, 22 May 2013

package capture

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

type Authorization interface {
	// adds credentials to the url, header, and form values to permit the request.
	Authorize(*url.URL, http.Header, url.Values) error
}

type AccessToken string

// adds an Authorize header containing the access token
func (token AccessToken) Authorize(uri *url.URL, header http.Header, values url.Values) error {
	header.Set("Authorization", "OAuth "+string(token))
	return nil
}

type ClientCredentialsSimple ClientCredentials

// adds id and secret as form post parameters.
func (creds *ClientCredentialsSimple) Authorize(uri *url.URL, header http.Header, values url.Values) error {
	values.Set("client_id", creds.Id)
	values.Set("client_secret", creds.Secret)
	return nil
}

type ClientCredentials struct {
	Id     string `json:"id" yaml:"id"`
	Secret string `json:"secret" yaml:"secret"`
}

// adds an Authorize header containing an HMAC-SHA1 signature
func (creds *ClientCredentials) Authorize(uri *url.URL, header http.Header, values url.Values) error {
	ps := make([]string, 0, len(values))
	for k := range values {
		for _, v := range values[k] {
			param := fmt.Sprintf("%s=%s", k, v) // not currently url encoded
			ps = append(ps, param)
		}
	}
	sort.Strings(ps)
	timestamp := time.Now().String()
	tosign := uri.Path
	tosign += "\n"
	tosign += timestamp
	tosign += "\n"
	for _, p := range ps {
		tosign += p
		tosign += "\n"
	}
	hash := hmac.New(sha1.New, []byte(creds.Secret))
	_, err := fmt.Fprint(hash, tosign)
	if err != nil {
		return err
	}
	sig := base64.URLEncoding.EncodeToString(hash.Sum(nil))
	header.Set("Date", timestamp)
	header.Set("Authorization", fmt.Sprintf("Signature %s:%s", creds.Id, sig))
	return nil
}
