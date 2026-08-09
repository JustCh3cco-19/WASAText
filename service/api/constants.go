package api

const (
	maxBinaryPayload   = 10 << 20
	maxMultipartMemory = 12 << 20
	maxRequestPayload  = 13 << 20
	maxJSONPayload     = 1 << 20
	sessionCookieName  = "wasatext_session"
	sessionMaxAge      = 24 * 60 * 60
)
