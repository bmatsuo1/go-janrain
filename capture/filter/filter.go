// Copyright 2013, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// filter.go [created: Thu, 23 May 2013]

/*
The filter package provides sereral helper methods for constructing filters used
to query entities with /entity.find and /entity.count. The package provides a
couple ways to create filters.

The easiest way to create a simple filter is to use the New() function. It
creates a filter constraining a single attribute relative to a given value.

	filter.New("displayName =", "chareth")),

The above function call produces a Filter type. Filters have chainable methods
And() and Or() which allow for building more complex filters.

	ageMin, ageMax = 18, 35
	now := time.Now()
	bdayMin, bdayMax = now.AddDate(-ageMax-1, 0, 1), now.AddDate(-ageMin, 0, 0)
	client.Execute("/entity.find", capture.Params{
		"filter": filter.
			New("gender =", "male").
			And("birthday >=", bdayMin).
			And("birthday <", bdayMax).
			And("emailVerified is not", nil),
	})

An application can define types that act as filters. These types can be combined
with logical operators.

	type Organization struct {
		name string
		code string
	}
	func (org Organization) Filter() string {
		return filter.New("organization =", org.code).Filter()
	}

	type MinAge int
	func (age MinAge) Filter() string {
		return filter.New("birthday <", time.Now().AddDate(-int(age), 0, 0)).Filter()
	}

	func CountEmployees(client *capture.Client, org *Organization, minage int) (int, error) {
		resp, err := client.Execute("/entity.count", capture.Params{
			"filter": filter.And(org, MinAge(minage)),
		})
		if err == nil {
			return resp.Get("total_count").MustInt(), nil
		}
		return 0, err
	}
*/
package filter

import (
	"github.com/bmatsuo1/go-janrain/capture"

	"fmt"
	"strings"
	"time"
)

// escape and quote string values for filters.
func FilterEscapedString(val string) string {
	escaped := strings.NewReplacer(`'`, `\'`, `\`, `\\`).Replace(val)
	return fmt.Sprintf("'%v'", escaped)
}

// types that implement Interface can be used as filters and joined with
// logical operators.
type Interface interface {
	Filter() string
}

func filterSep(filters []Interface, sep string) Filter {
	switch len(filters) {
	case 0:
		return ""
	case 1:
		return Filter(filters[0].Filter())
	}

	_filters := make([]string, len(filters))
	for i := range filters {
		_filters[i] = fmt.Sprintf("(%s)", filters[i].Filter())
	}
	filter := strings.Join(_filters, sep)
	return Filter(filter)
}

// join multiple filters in a conjunction
func And(filters ...Interface) Filter {
	return filterSep(filters, "AND")
}

// join multiple filters in a disjunction
func Or(filters ...Interface) Filter {
	return filterSep(filters, "OR")
}

// an attribute constraint described by FilterStr and relative to Value
type F struct {
	FilterStr string
	Value     interface{}
}

// a filter value that implements Value has it's FilterValue() string inserted
// directly into constructed filter strings without being escaped.
type Value interface {
	FilterValue() string
}

type timeValue time.Time

func (t timeValue) FilterValue() string {
	return fmt.Sprintf("'%v'", capture.Timestamp(time.Time(t)))
}

type stringValue string

func (s stringValue) FilterValue() string {
	return fmt.Sprintf("%s", FilterEscapedString(string(s)))
}

type arbitraryValue struct {
	val interface{}
}

func (v arbitraryValue) FilterValue() string {
	return fmt.Sprintf("%v", v.val)
}

func (c *F) Filter() string {
	var val Value
	switch c.Value.(type) {
	case Value:
		val = c.Value.(Value)
	case string:
		val = stringValue(c.Value.(string))
	case time.Time:
		val = timeValue(c.Value.(time.Time))
	default:
		val = arbitraryValue{c.Value}
	}
	return fmt.Sprintf("%s %s", c.FilterStr, val.FilterValue())
}

func (c *F) String() string {
	return c.Filter()
}

type Filter string

// a helper function for constructing filters for the /entity.find API call.
func New(attrcomp string, val interface{}) Filter {
	return Filter((&F{attrcomp, val}).String())
}

// add an additional constraint to the filter.
func (filter Filter) And(attrcomp string, val interface{}) Filter {
	return Filter(fmt.Sprintf("(%s) AND (%s)", filter, New(attrcomp, val)))
}

// add an alternative constraint to filter.
func (filter Filter) Or(attrcomp string, val interface{}) Filter {
	return Filter(fmt.Sprintf("(%s) OR (%s)", filter, New(attrcomp, val)))
}

// implements Interface
func (filter Filter) Filter() string {
	return string(filter)
}
