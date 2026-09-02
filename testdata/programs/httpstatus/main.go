package main

import (
	"fmt"
	"net/http"
)

// codes holds each status code that the IANA registry names.
var codes = []int{
	http.StatusContinue, http.StatusSwitchingProtocols, http.StatusProcessing,
	http.StatusEarlyHints,
	http.StatusOK, http.StatusCreated, http.StatusAccepted,
	http.StatusNonAuthoritativeInfo, http.StatusNoContent,
	http.StatusResetContent, http.StatusPartialContent, http.StatusMultiStatus,
	http.StatusAlreadyReported, http.StatusIMUsed,
	http.StatusMultipleChoices, http.StatusMovedPermanently, http.StatusFound,
	http.StatusSeeOther, http.StatusNotModified, http.StatusUseProxy,
	http.StatusTemporaryRedirect, http.StatusPermanentRedirect,
	http.StatusBadRequest, http.StatusUnauthorized, http.StatusPaymentRequired,
	http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed,
	http.StatusNotAcceptable, http.StatusProxyAuthRequired,
	http.StatusRequestTimeout, http.StatusConflict, http.StatusGone,
	http.StatusLengthRequired, http.StatusPreconditionFailed,
	http.StatusRequestEntityTooLarge, http.StatusRequestURITooLong,
	http.StatusUnsupportedMediaType, http.StatusRequestedRangeNotSatisfiable,
	http.StatusExpectationFailed, http.StatusTeapot,
	http.StatusMisdirectedRequest, http.StatusUnprocessableEntity,
	http.StatusLocked, http.StatusFailedDependency, http.StatusTooEarly,
	http.StatusUpgradeRequired, http.StatusPreconditionRequired,
	http.StatusTooManyRequests, http.StatusRequestHeaderFieldsTooLarge,
	http.StatusUnavailableForLegalReasons,
	http.StatusInternalServerError, http.StatusNotImplemented,
	http.StatusBadGateway, http.StatusServiceUnavailable,
	http.StatusGatewayTimeout, http.StatusHTTPVersionNotSupported,
	http.StatusVariantAlsoNegotiates, http.StatusInsufficientStorage,
	http.StatusLoopDetected, http.StatusNotExtended,
	http.StatusNetworkAuthenticationRequired,
}

func main() {
	for _, code := range codes {
		fmt.Printf("%d %s\n", code, http.StatusText(code))
	}
	fmt.Printf("%d codes\n", len(codes))
	fmt.Printf("an unknown code gives %q\n", http.StatusText(299))
}
