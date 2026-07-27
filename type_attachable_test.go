package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestAttachableServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/attachable/5000000000000010341", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Attachable":{"Id":"5000000000000010341","SyncToken":"0","domain":"QBO","Note":"This is an attached note."}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	att, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Attachables().Get(context.Background(), "5000000000000010341")
	require.NoError(t, err)
	assert.Equal(t, "5000000000000010341", att.ID())
	assert.Equal(t, EntityAttachable, att.EntityName())
	assert.Equal(t, "This is an attached note.", att.Note)
}

func TestAttachableServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Attachable STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Attachable":[{"Id":"10341","SyncToken":"0"},{"Id":"10342","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	atts, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Attachables().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, atts, 2)
	assert.Equal(t, "10341", atts[0].ID())
	assert.Equal(t, "10342", atts[1].ID())
}

func TestAttachableJSONDecodesNoteAttachment(t *testing.T) {
	const fixture = `{
		"Attachable": {
			"Id": "5000000000000010341",
			"SyncToken": "0",
			"MetaData": {
				"CreateTime": "2015-11-17T11:05:15-08:00",
				"LastUpdatedTime": "2015-11-17T11:05:15-08:00"
			},
			"Note": "This is an attached note.",
			"Tag": "expense-q4",
			"AttachableRef": [
				{
					"IncludeOnSend": false,
					"EntityRef": {"type": "Invoice", "value": "95"}
				}
			]
		}
	}`

	var envelope struct {
		Attachable Attachable `json:"Attachable"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	att := envelope.Attachable
	assert.Equal(t, "5000000000000010341", att.ID())
	assert.Equal(t, "0", att.SyncToken())

	require.NotNil(t, att.MetaData)
	require.NotNil(t, att.MetaData.CreateTime)

	assert.Equal(t, "This is an attached note.", att.Note)
	assert.Equal(t, "expense-q4", att.Tag)
	assert.Empty(t, att.FileName)
	assert.Empty(t, att.FileAccessUri)

	require.Len(t, att.AttachableRef, 1)
	ref := att.AttachableRef[0]
	require.NotNil(t, ref.IncludeOnSend)
	assert.False(t, *ref.IncludeOnSend)
	require.NotNil(t, ref.EntityRef)
	assert.Equal(t, "Invoice", ref.EntityRef.Type)
	assert.Equal(t, "95", ref.EntityRef.Value)
}

func TestAttachableJSONDecodesFileAttachment(t *testing.T) {
	const fixture = `{
		"Attachable": {
			"Id": "100000000004190865",
			"SyncToken": "0",
			"MetaData": {
				"CreateTime": "2015-11-16T10:59:02-08:00",
				"LastUpdatedTime": "2015-11-16T10:59:02-08:00"
			},
			"FileName": "receipt_nov15.jpg",
			"ContentType": "image/jpeg",
			"Category": "Receipt",
			"Size": 1594261,
			"FileAccessUri": "https://qbo.intuit.com/qbo39/files/...",
			"ThumbnailFileAccessUri": "https://qbo.intuit.com/qbo39/files/thumb/...",
			"TempDownloadUri": "https://s3.amazonaws.com/...?Expires=123",
			"ThumbnailTempDownloadUri": "https://s3.amazonaws.com/.../thumb?Expires=123",
			"AttachableRef": [
				{
					"IncludeOnSend": false,
					"EntityRef": {"type": "Invoice", "value": "95"}
				}
			]
		}
	}`

	var envelope struct {
		Attachable Attachable `json:"Attachable"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	att := envelope.Attachable
	assert.Equal(t, "100000000004190865", att.ID())
	assert.Equal(t, "receipt_nov15.jpg", att.FileName)
	assert.Equal(t, ContentTypeJPEG, att.ContentType)
	assert.Equal(t, ReceiptAttachableCategory, att.Category)
	assert.Equal(t, "1594261", att.Size.String())
	assert.Equal(t, "https://qbo.intuit.com/qbo39/files/...", att.FileAccessUri)
	assert.Equal(t, "https://qbo.intuit.com/qbo39/files/thumb/...", att.ThumbnailFileAccessUri)
	assert.Equal(t, "https://s3.amazonaws.com/...?Expires=123", att.TempDownloadUri)
	assert.Equal(t, "https://s3.amazonaws.com/.../thumb?Expires=123", att.ThumbnailTempDownloadUri)
	assert.Empty(t, att.Note)
}

func TestAttachableRequestsMarshalWritablePayloads(t *testing.T) {
	create := Attachable{
		Note: "This is an attached note.",
		AttachableRef: []AttachableRef{{
			EntityRef:     &ReferenceType{Type: "Invoice", Value: "95"},
			IncludeOnSend: boolPtr(false),
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Note"`)
	assert.Contains(t, string(body), `"AttachableRef"`)
	assert.NotContains(t, string(body), `"FileAccessUri"`)
	assert.NotContains(t, string(body), `"TempDownloadUri"`)
	assert.NotContains(t, string(body), `"Size"`)
}

func TestAttachableDeleteUsesCorrectEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/attachable", r.URL.Path)
		assert.Equal(t, "delete", r.URL.Query().Get("operation"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Attachable":{"Id":"10341","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	result, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Attachables().Delete(
		context.Background(),
		DeleteRequest{IDValue: IDValue{Id: "10341"}, SyncTokenValue: SyncTokenValue{SyncTokenValue: "0"}},
	)
	require.NoError(t, err)
	assert.Equal(t, "10341", result.ID())
}

func TestAttachableDownloadReturnsURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/download/10341", r.URL.Path)
		w.Header().Set("Content-Type", "text/plain")
		// QBO returns the pre-signed URL as plain text.
		_, err := io.WriteString(w, `https://s3.amazonaws.com/attachments/receipt.pdf?Expires=123&Signature=abc`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	downloadURL, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Attachables().Download(context.Background(), "10341")
	require.NoError(t, err)
	assert.Equal(t, "https://s3.amazonaws.com/attachments/receipt.pdf?Expires=123&Signature=abc", downloadURL)
}

func TestAttachableUploadBuildsMultipartRequest(t *testing.T) {
	var capturedContentType string
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/upload", r.URL.Path)
		capturedContentType = r.Header.Get("Content-Type")
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"AttachableResponse":[{"Attachable":{"Id":"100000000004190865","SyncToken":"0","FileName":"receipt.jpg","ContentType":"image/jpeg","Size":42}}]}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	meta := Attachable{
		FileName:    "receipt.jpg",
		ContentType: "image/jpeg",
		AttachableRef: []AttachableRef{{
			EntityRef: &ReferenceType{Type: "Invoice", Value: "95"},
		}},
	}

	result, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Attachables().Upload(context.Background(), UploadFile{Meta: meta, Content: strings.NewReader("fake-jpeg-bytes")})
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, "100000000004190865", result[0].ID())
	assert.Equal(t, "receipt.jpg", result[0].FileName)
	assert.Equal(t, "42", result[0].Size.String())

	// Verify the request had the correct multipart content type.
	assert.Contains(t, capturedContentType, "multipart/form-data")
	assert.Contains(t, capturedContentType, "boundary=")

	// Verify both the file content and metadata JSON appear in the body.
	bodyStr := string(capturedBody)
	assert.Contains(t, bodyStr, "fake-jpeg-bytes")
	assert.Contains(t, bodyStr, `"FileName"`)
	assert.Contains(t, bodyStr, `"AttachableRef"`)
}

func TestAttachableUploadMultipleFiles(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"AttachableResponse":[{"Attachable":{"Id":"1","FileName":"a.pdf"}},{"Attachable":{"Id":"2","FileName":"b.png"}}]}`)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	result, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "t"})).Attachables().Upload(
		context.Background(),
		UploadFile{Meta: Attachable{FileName: "a.pdf"}, Content: strings.NewReader("PDF-BYTES")},
		UploadFile{Meta: Attachable{FileName: "b.png"}, Content: strings.NewReader("PNG-BYTES")},
	)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "1", result[0].ID())
	assert.Equal(t, "2", result[1].ID())

	// Each file gets a numbered content/metadata part pair.
	body := string(capturedBody)
	assert.Contains(t, body, `name="file_content_01"`)
	assert.Contains(t, body, `name="file_metadata_01"`)
	assert.Contains(t, body, `name="file_content_02"`)
	assert.Contains(t, body, `name="file_metadata_02"`)
	// ContentType is normalised from the file extension.
	assert.Contains(t, body, `"ContentType":"application/pdf"`)
	assert.Contains(t, body, `"ContentType":"image/png"`)
}

func TestAttachableUploadRejectsUnsupportedType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called for an unsupported type")
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "t"})).Attachables().Upload(
		context.Background(),
		UploadFile{Meta: Attachable{FileName: "icon.svg"}, Content: strings.NewReader("<svg/>")},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestAttachableUploadRejectsOversizeFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called for an oversize file")
	}))
	defer server.Close()

	orig := MaxAttachmentFileSize
	MaxAttachmentFileSize = 8
	defer func() { MaxAttachmentFileSize = orig }()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "t"})).Attachables().Upload(
		context.Background(),
		UploadFile{Meta: Attachable{FileName: "big.txt"}, Content: strings.NewReader("0123456789")}, // 10 bytes > 8
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "per-file limit")
}

func TestAttachableUploadErrorsOnEmptyResponseEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// An entry with no Attachable (empty Id) must not be treated as success.
		_, _ = io.WriteString(w, `{"AttachableResponse":[{"Fault":{"type":"ValidationFault","Error":[{"Message":"rejected"}]}}]}`)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "t"})).Attachables().Upload(
		context.Background(),
		UploadFile{Meta: Attachable{FileName: "a.pdf"}, Content: strings.NewReader("x")},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no attachable")
}

func TestAttachableGetFileDownloadsAndFetches(t *testing.T) {
	fileBytes := []byte("fake-pdf-bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/company/123/download/10341":
			// Return the pre-signed URL pointing back at this same test server.
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "http://%s/files/receipt.pdf", r.Host)
		case "/files/receipt.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, err := w.Write(fileBytes)
			require.NoError(t, err)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	got, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Attachables().GetFile(context.Background(), "10341")
	require.NoError(t, err)
	assert.Equal(t, fileBytes, got)
}

func TestAttachableStreamFileDownloadsAndStreams(t *testing.T) {
	fileBytes := []byte("streamed-file-content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/company/123/download/10342":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "http://%s/files/image.jpg", r.Host)
		case "/files/image.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, err := w.Write(fileBytes)
			require.NoError(t, err)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Attachables().StreamFile(context.Background(), "10342")
	require.NoError(t, err)
	require.NotNil(t, rc)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, fileBytes, got)
}

// Compile-time interface compliance checks for Attachable and AttachableService.
var (
	_ Entity    = Attachable{}
	_ CDCEntity = Attachable{}

	_ Getter[Attachable]          = (*AttachableService)(nil)
	_ Querier[Attachable]         = (*AttachableService)(nil)
	_ Lister[Attachable]          = (*AttachableService)(nil)
	_ Pager[Attachable]           = (*AttachableService)(nil)
	_ Creator[Attachable]         = (*AttachableService)(nil)
	_ Updater[Attachable]         = (*AttachableService)(nil)
	_ SparseUpdater[Attachable]   = (*AttachableService)(nil)
	_ Deleter[Attachable]         = (*AttachableService)(nil)
	_ FileGetter                  = (*AttachableService)(nil)
	_ FileStreamer                = (*AttachableService)(nil)
	_ EntityCDCReader[Attachable] = (*AttachableService)(nil)
)

func TestAttachableServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/attachable", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"AttachableResponse":[{"Attachable":{"Id":"1","SyncToken":"2"}}]}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	a := Attachable{}
	a.Id = "1"
	a.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Attachables().UpdateSparse(context.Background(), a)
	require.NoError(t, err)
}
