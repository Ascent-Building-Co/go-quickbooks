package quickbooks

import (
	"path"
	"sort"
	"strings"
)

// AttachmentContentType is a MIME content type accepted by the QuickBooks
// Online upload endpoint. QBO restricts attachments to a fixed set of file
// types; uploads are validated by file extension and the metadata's
// ContentType is normalised to the canonical value for that extension.
//
// The canonical values are the industry-standard MIME types for each
// extension (the QBO API docs list a few non-standard variants, e.g.
// "image/jpg" or "image/tif", which real clients rarely send).
type AttachmentContentType string

const (
	ContentTypePostscript AttachmentContentType = "application/postscript" // .ai, .eps
	ContentTypeCSV        AttachmentContentType = "text/csv"
	ContentTypeMSWord     AttachmentContentType = "application/msword"
	ContentTypeDOCX       AttachmentContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ContentTypeGIF        AttachmentContentType = "image/gif"
	ContentTypeJPEG       AttachmentContentType = "image/jpeg"
	ContentTypeODS        AttachmentContentType = "application/vnd.oasis.opendocument.spreadsheet"
	ContentTypePDF        AttachmentContentType = "application/pdf"
	ContentTypePNG        AttachmentContentType = "image/png"
	ContentTypeRTF        AttachmentContentType = "application/rtf"
	ContentTypeTIFF       AttachmentContentType = "image/tiff"
	ContentTypeText       AttachmentContentType = "text/plain"
	ContentTypeMSExcel    AttachmentContentType = "application/vnd.ms-excel"
	ContentTypeXLSX       AttachmentContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	ContentTypeXML        AttachmentContentType = "application/xml"
)

// approvedAttachmentTypes maps each approved lowercase file extension (without
// the dot) to its canonical content type. Multiple extensions may map to the
// same type. This is the authoritative set of file types QBO accepts.
var approvedAttachmentTypes = map[string]AttachmentContentType{
	"ai":   ContentTypePostscript,
	"eps":  ContentTypePostscript,
	"csv":  ContentTypeCSV,
	"doc":  ContentTypeMSWord,
	"docx": ContentTypeDOCX,
	"gif":  ContentTypeGIF,
	"jpg":  ContentTypeJPEG,
	"jpeg": ContentTypeJPEG,
	"ods":  ContentTypeODS,
	"pdf":  ContentTypePDF,
	"png":  ContentTypePNG,
	"rtf":  ContentTypeRTF,
	"tif":  ContentTypeTIFF,
	"txt":  ContentTypeText,
	"xls":  ContentTypeMSExcel,
	"xlsx": ContentTypeXLSX,
	"xml":  ContentTypeXML,
}

// AttachmentTypeForFilename returns the canonical content type for the file
// name's extension and whether that extension is approved for upload. The
// extension match is case-insensitive.
func AttachmentTypeForFilename(filename string) (AttachmentContentType, bool) {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(filename), "."))
	ct, ok := approvedAttachmentTypes[ext]
	return ct, ok
}

// ApprovedAttachmentExtensions returns the sorted list of approved upload file
// extensions, suitable for user-facing error messages.
func ApprovedAttachmentExtensions() []string {
	exts := make([]string, 0, len(approvedAttachmentTypes))
	for ext := range approvedAttachmentTypes {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}
