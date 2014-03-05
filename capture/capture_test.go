package capture

import (
	"fmt"
	"testing"
)

func TestErrors(t *testing.T) {
	var jsonerr error = JsonDecoderError{fmt.Errorf("boo!")} // fits interface
	if jsonerr.Error() != "boo!" {
		fmt.Errorf("unexpected json decoder error message: %q", jsonerr.Error())
	}
	var ctypeerr error = ContentTypeError("...")
	if ctypeerr.Error() != "..." {
		fmt.Errorf("unexpected content-type error message: %q", ctypeerr.Error())
	}
	var httperr error = HttpTransportError{fmt.Errorf("oop--")}
	if httperr.Error() != "oop--" {
		fmt.Errorf("unexpected http transport error message: %q", httperr.Error())
	}
	var remerr error = RemoteError{Kind: "bad", Description: "err msg"}
	if remerr.Error() != "bad [0] err msg (response: null)" {
		fmt.Errorf("unexpected remote error message: %q", remerr.Error())
	}
}
