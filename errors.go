package quickbooks

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrOAuthConfigNotSet = errors.New("oauth config is not configured")
	ErrRealmIDRequired   = errors.New("realm id is required")
	ErrTokenSourceNil    = errors.New("token source is required")
)

type Fault struct {
	Error []FaultError `json:"Error" xml:"Error"`
	Type  string       `json:"type" xml:"type,attr"`
}

type FaultError struct {
	Message string `json:"Message" xml:"Message"`
	Detail  string `json:"Detail" xml:"Detail"`
	Code    string `json:"code" xml:"code,attr"`
	Element string `json:"element" xml:"element,attr"`
}

type Failure struct {
	Fault Fault  `json:"Fault" xml:"Fault"`
	Time  string `json:"time" xml:"time,attr"`
}

func (f *Failure) HasErrors() bool {
	return f != nil && len(f.Fault.Error) > 0
}

// APIError is returned when the QBO API responds with a non-2xx status. It
// carries the HTTP status code, the raw response body, and — when the API
// returns a structured Intuit fault — a parsed Failure. Use errors.As to
// inspect it from wrapped errors.
//
// IntuitTID is the Intuit transaction ID from the response headers, useful
// for support requests.
type APIError struct {
	StatusCode int
	Body       string
	Failure    *Failure
	Headers    http.Header
	IntuitTID  string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	text := fmt.Sprintf("quickbooks API error: status %d", e.StatusCode)

	if messages := e.messages(); len(messages) > 0 {
		text = fmt.Sprintf("%s: %s", text, strings.Join(messages, "; "))
	} else if e.Body != "" {
		text = fmt.Sprintf("%s: %s", text, e.Body)
	}

	if e.IntuitTID != "" {
		text = fmt.Sprintf("%s (intuit_tid %s)", text, e.IntuitTID)
	}

	return text
}

func (e *APIError) messages() []string {
	if e == nil || e.Failure == nil {
		return nil
	}

	messages := make([]string, 0, len(e.Failure.Fault.Error))
	for _, item := range e.Failure.Fault.Error {
		switch {
		case item.Detail != "":
			messages = append(messages, item.Detail)
		case item.Message != "":
			messages = append(messages, item.Message)
		case item.Code != "":
			messages = append(messages, "error code "+item.Code)
		}
	}

	return messages
}

// RateLimitError is returned when the QBO API responds with HTTP 429. It
// embeds APIError and adds realm, request ID, and Retry-After header for
// diagnostics. Use errors.As to distinguish it from other APIErrors.
type RateLimitError struct {
	APIError
	RealmID    string
	RequestID  string
	RetryAfter string
}

func (e RateLimitError) Error() string {
	parts := []string{fmt.Sprintf("rate limit error: status %d", e.StatusCode)}
	if messages := e.messages(); len(messages) > 0 {
		parts = append(parts, strings.Join(messages, "; "))
	}
	if e.RealmID != "" {
		parts = append(parts, "realm "+e.RealmID)
	}
	if e.RequestID != "" {
		parts = append(parts, "request id "+e.RequestID)
	}
	if e.RetryAfter != "" {
		parts = append(parts, "retry after "+e.RetryAfter)
	}
	if e.IntuitTID != "" {
		parts = append(parts, "intuit_tid "+e.IntuitTID)
	}
	return strings.Join(parts, ", ")
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	reader, err := responseReader(resp)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func newAPIError(statusCode int, headers http.Header, body []byte) *APIError {
	bodyText := strings.TrimSpace(string(body))

	err := &APIError{
		StatusCode: statusCode,
		Body:       bodyText,
		Headers:    cloneHeader(headers),
		IntuitTID:  headers.Get("intuit_tid"),
	}

	if failure := parseFailureBody(body); failure != nil {
		err.Failure = failure
	}

	return err
}

// parseFailureBody parses the fault in body: a fault that is the body
// itself, or else the faults of a batch response's items, flattened into one.
func parseFailureBody(body []byte) *Failure {
	return parseFailure(body, true)
}

// parseBodyFault parses a fault that is body itself, ignoring the faults of
// a batch response's items.
func parseBodyFault(body []byte) *Failure {
	return parseFailure(body, false)
}

func parseFailure(body []byte, batchItems bool) *Failure {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}

	if failure := parseJSONFailure(body, batchItems); failure != nil {
		return failure
	}

	if failure := parseXMLFailure(body, batchItems); failure != nil {
		return failure
	}

	return nil
}

func parseJSONFailure(body []byte, batchItems bool) *Failure {
	var failure Failure
	if err := json.Unmarshal(body, &failure); err == nil && failure.HasErrors() {
		return &failure
	}
	if !batchItems {
		return nil
	}

	var batch struct {
		Time              string `json:"time"`
		BatchItemResponse []struct {
			Fault Fault `json:"Fault"`
		} `json:"BatchItemResponse"`
		BatchItemResp []struct {
			Fault Fault `json:"Fault"`
		} `json:"BatchItemResp"`
	}
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil
	}

	faults := make([]Fault, 0, len(batch.BatchItemResponse)+len(batch.BatchItemResp))
	for _, item := range batch.BatchItemResponse {
		faults = append(faults, item.Fault)
	}
	for _, item := range batch.BatchItemResp {
		faults = append(faults, item.Fault)
	}

	if failure := flattenBatchFailure(batch.Time, faults); failure != nil {
		return failure
	}

	return nil
}

func parseXMLFailure(body []byte, batchItems bool) *Failure {
	var failure Failure
	if err := xml.Unmarshal(body, &failure); err == nil && failure.HasErrors() {
		return &failure
	}

	var batch struct {
		XMLName           xml.Name `xml:"IntuitResponse"`
		Time              string   `xml:"time,attr"`
		BatchItemResponse []struct {
			Fault Fault `xml:"Fault"`
		} `xml:"BatchItemResponse"`
		BatchItemResp []struct {
			Fault Fault `xml:"Fault"`
		} `xml:"BatchItemResp"`
	}
	if err := xml.Unmarshal(body, &batch); batchItems && err == nil {
		faults := make([]Fault, 0, len(batch.BatchItemResponse)+len(batch.BatchItemResp))
		for _, item := range batch.BatchItemResponse {
			faults = append(faults, item.Fault)
		}
		for _, item := range batch.BatchItemResp {
			faults = append(faults, item.Fault)
		}

		if failure := flattenBatchFailure(batch.Time, faults); failure != nil {
			return failure
		}
	}

	var fault Fault
	if err := xml.Unmarshal(body, &fault); err == nil && len(fault.Error) > 0 {
		return &Failure{Fault: fault}
	}

	return nil
}

func flattenBatchFailure(time string, faults []Fault) *Failure {
	combined := Failure{Time: time}
	for _, fault := range faults {
		if len(fault.Error) == 0 {
			continue
		}
		if combined.Fault.Type == "" {
			combined.Fault.Type = fault.Type
		}
		combined.Fault.Error = append(combined.Fault.Error, fault.Error...)
	}

	if !combined.HasErrors() {
		return nil
	}

	return &combined
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}

	clone := make(http.Header, len(header))
	for key, values := range header {
		clone[key] = append([]string(nil), values...)
	}

	return clone
}
