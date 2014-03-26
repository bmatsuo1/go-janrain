package capture

import (
	"fmt"
	"net/http"
	"testing"
)

func TestErrors(t *testing.T) {
	var jsonerr error = JsonDecoderError{nil, fmt.Errorf("boo!")} // fits interface
	if jsonerr.Error() != "boo!" {
		t.Errorf("unexpected json decoder error message: %q", jsonerr.Error())
	}
	var ctypeerr error = &ContentTypeError{
		&HttpResponseData{
			Header: http.Header{"Content-Type": {"application/goboom"}},
		},
	}
	if ctypeerr.Error() != `unexpected content-type "application/goboom"` {
		t.Errorf("unexpected content-type error message: %q", ctypeerr.Error())
	}
	var httperr error = HttpTransportError{fmt.Errorf("oop--")}
	if httperr.Error() != "oop--" {
		t.Errorf("unexpected http transport error message: %q", httperr.Error())
	}
	var remerr error = RemoteError{Kind: "invalid_kittens", Description: "nyan nyan"}
	if remerr.Error() != "[invalid_kittens] nyan nyan" {
		t.Errorf("unexpected remote error message: %q", remerr.Error())
	}
}
