package httpmiddleware

const (
	FieldType           = "type"
	FieldURL            = "url_path"
	FieldReqHeader      = "req_header"
	FieldReqBody        = "req_body"
	FieldResponseHeader = "rsp_header"
	FieldStatus         = "status"
	FieldResponseBody   = "rsp_body"
	FieldDurationMs     = "duration_ms"
	FieldReqTimestamp   = "req_timestamp"
)

const (
	ExcludeLog = true
	IncludeLog = false
)

const (
	headerNameRequestID = "x-request-id"

	EventPrefix  = "events"
	URLSeparator = "/"
)

const (
	wipedMessage = "-"
)
