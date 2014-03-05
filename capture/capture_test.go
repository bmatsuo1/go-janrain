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
}
