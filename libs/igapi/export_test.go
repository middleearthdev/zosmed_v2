package igapi

// export_test.go exposes test-only seams for the igapi_test package. It is
// excluded from production builds (Go only compiles _test.go files during
// `go test`). This is the standard Go idiom for letting external tests
// override unexported package state without widening the public API.

// SetOAuthHostsForTest points the four OAuth endpoint hosts at test doubles
// (e.g. an httptest.Server) for the duration of a test, and returns a
// restore function that must be deferred to put the real hosts back.
func SetOAuthHostsForTest(authorize, exchangeCode, exchangeLongLived, refreshLongLived string) (restore func()) {
	prevAuthorize := oauthAuthorizeURL
	prevExchangeCode := oauthExchangeCodeURL
	prevExchangeLongLived := oauthExchangeLongLivedURL
	prevRefreshLongLived := oauthRefreshLongLivedURL

	oauthAuthorizeURL = authorize
	oauthExchangeCodeURL = exchangeCode
	oauthExchangeLongLivedURL = exchangeLongLived
	oauthRefreshLongLivedURL = refreshLongLived

	return func() {
		oauthAuthorizeURL = prevAuthorize
		oauthExchangeCodeURL = prevExchangeCode
		oauthExchangeLongLivedURL = prevExchangeLongLived
		oauthRefreshLongLivedURL = prevRefreshLongLived
	}
}
