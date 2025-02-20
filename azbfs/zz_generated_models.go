package azbfs

// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"encoding/base64"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// concatenates a slice of const values with the specified separator between each item
func joinConst(s interface{}, sep string) string {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		panic("s wasn't a slice or array")
	}
	ss := make([]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		ss = append(ss, v.Index(i).String())
	}
	return strings.Join(ss, sep)
}

func validateError(err error) {
	if err != nil {
		panic(err)
	}
}

// PathGetPropertiesActionType enumerates the values for path get properties action type.
type PathGetPropertiesActionType string

const (
	// PathGetPropertiesActionGetAccessControl ...
	PathGetPropertiesActionGetAccessControl PathGetPropertiesActionType = "getAccessControl"
	// PathGetPropertiesActionGetStatus ...
	PathGetPropertiesActionGetStatus PathGetPropertiesActionType = "getStatus"
	// PathGetPropertiesActionNone represents an empty PathGetPropertiesActionType.
	PathGetPropertiesActionNone PathGetPropertiesActionType = ""
)

// PossiblePathGetPropertiesActionTypeValues returns an array of possible values for the PathGetPropertiesActionType const type.
func PossiblePathGetPropertiesActionTypeValues() []PathGetPropertiesActionType {
	return []PathGetPropertiesActionType{PathGetPropertiesActionGetAccessControl, PathGetPropertiesActionGetStatus, PathGetPropertiesActionNone}
}

// Marker represents an opaque value used in paged responses.
type Marker struct {
	Val *string
}

// NotDone returns true if the list enumeration should be started or is not yet complete. Specifically, NotDone returns true
// for a just-initialized (zero value) Marker indicating that you should make an initial request to get a result portion from
// the service. NotDone also returns true whenever the service returns an interim result portion. NotDone returns false only
// after the service has returned the final result portion.
func (m Marker) NotDone() bool {
	return m.Val == nil || *m.Val != ""
}

// PathLeaseActionType enumerates the values for path lease action type.
type PathLeaseActionType string

const (
	// PathLeaseActionAcquire ...
	PathLeaseActionAcquire PathLeaseActionType = "acquire"
	// PathLeaseActionBreak ...
	PathLeaseActionBreak PathLeaseActionType = "break"
	// PathLeaseActionChange ...
	PathLeaseActionChange PathLeaseActionType = "change"
	// PathLeaseActionNone represents an empty PathLeaseActionType.
	PathLeaseActionNone PathLeaseActionType = ""
	// PathLeaseActionRelease ...
	PathLeaseActionRelease PathLeaseActionType = "release"
	// PathLeaseActionRenew ...
	PathLeaseActionRenew PathLeaseActionType = "renew"
)

// PossiblePathLeaseActionTypeValues returns an array of possible values for the PathLeaseActionType const type.
func PossiblePathLeaseActionTypeValues() []PathLeaseActionType {
	return []PathLeaseActionType{PathLeaseActionAcquire, PathLeaseActionBreak, PathLeaseActionChange, PathLeaseActionNone, PathLeaseActionRelease, PathLeaseActionRenew}
}

// PathRenameModeType enumerates the values for path rename mode type.
type PathRenameModeType string

const (
	// PathRenameModeLegacy ...
	PathRenameModeLegacy PathRenameModeType = "legacy"
	// PathRenameModeNone represents an empty PathRenameModeType.
	PathRenameModeNone PathRenameModeType = ""
	// PathRenameModePosix ...
	PathRenameModePosix PathRenameModeType = "posix"
)

// PossiblePathRenameModeTypeValues returns an array of possible values for the PathRenameModeType const type.
func PossiblePathRenameModeTypeValues() []PathRenameModeType {
	return []PathRenameModeType{PathRenameModeLegacy, PathRenameModeNone, PathRenameModePosix}
}

// PathResourceType enumerates the values for path resource type.
type PathResourceType string

const (
	// PathResourceDirectory ...
	PathResourceDirectory PathResourceType = "directory"
	// PathResourceFile ...
	PathResourceFile PathResourceType = "file"
	// PathResourceNone represents an empty PathResourceType.
	PathResourceNone PathResourceType = ""
)

// PossiblePathResourceTypeValues returns an array of possible values for the PathResourceType const type.
func PossiblePathResourceTypeValues() []PathResourceType {
	return []PathResourceType{PathResourceDirectory, PathResourceFile, PathResourceNone}
}

// PathUpdateActionType enumerates the values for path update action type.
type PathUpdateActionType string

const (
	// PathUpdateActionAppend ...
	PathUpdateActionAppend PathUpdateActionType = "append"
	// PathUpdateActionFlush ...
	PathUpdateActionFlush PathUpdateActionType = "flush"
	// PathUpdateActionNone represents an empty PathUpdateActionType.
	PathUpdateActionNone PathUpdateActionType = ""
	// PathUpdateActionSetAccessControl ...
	PathUpdateActionSetAccessControl PathUpdateActionType = "setAccessControl"
	// PathUpdateActionSetProperties ...
	PathUpdateActionSetProperties PathUpdateActionType = "setProperties"
)

// PossiblePathUpdateActionTypeValues returns an array of possible values for the PathUpdateActionType const type.
func PossiblePathUpdateActionTypeValues() []PathUpdateActionType {
	return []PathUpdateActionType{PathUpdateActionAppend, PathUpdateActionFlush, PathUpdateActionNone, PathUpdateActionSetAccessControl, PathUpdateActionSetProperties}
}

// DataLakeStorageError ...
type DataLakeStorageError struct {
	// Error - The service error response object.
	Error *DataLakeStorageErrorError `json:"error,omitempty"`
}

// DataLakeStorageErrorError - The service error response object.
type DataLakeStorageErrorError struct {
	// Code - The service error code.
	Code *string `json:"code,omitempty"`
	// Message - The service error message.
	Message *string `json:"message,omitempty"`
}

// Filesystem ...
type Filesystem struct {
	Name         *string `json:"name,omitempty"`
	LastModified *string `json:"lastModified,omitempty"`
	ETag         *string `json:"eTag,omitempty"`
}

// FilesystemCreateResponse ...
type FilesystemCreateResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (fcr FilesystemCreateResponse) Response() *http.Response {
	return fcr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (fcr FilesystemCreateResponse) StatusCode() int {
	return fcr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (fcr FilesystemCreateResponse) Status() string {
	return fcr.rawResponse.Status
}

// Date returns the value for header Date.
func (fcr FilesystemCreateResponse) Date() string {
	return fcr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (fcr FilesystemCreateResponse) ETag() string {
	return fcr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (fcr FilesystemCreateResponse) LastModified() string {
	return fcr.rawResponse.Header.Get("Last-Modified")
}

// XMsNamespaceEnabled returns the value for header x-ms-namespace-enabled.
func (fcr FilesystemCreateResponse) XMsNamespaceEnabled() string {
	return fcr.rawResponse.Header.Get("x-ms-namespace-enabled")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (fcr FilesystemCreateResponse) XMsRequestID() string {
	return fcr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (fcr FilesystemCreateResponse) XMsVersion() string {
	return fcr.rawResponse.Header.Get("x-ms-version")
}

// FilesystemDeleteResponse ...
type FilesystemDeleteResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (fdr FilesystemDeleteResponse) Response() *http.Response {
	return fdr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (fdr FilesystemDeleteResponse) StatusCode() int {
	return fdr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (fdr FilesystemDeleteResponse) Status() string {
	return fdr.rawResponse.Status
}

// Date returns the value for header Date.
func (fdr FilesystemDeleteResponse) Date() string {
	return fdr.rawResponse.Header.Get("Date")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (fdr FilesystemDeleteResponse) XMsRequestID() string {
	return fdr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (fdr FilesystemDeleteResponse) XMsVersion() string {
	return fdr.rawResponse.Header.Get("x-ms-version")
}

// FilesystemGetPropertiesResponse ...
type FilesystemGetPropertiesResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (fgpr FilesystemGetPropertiesResponse) Response() *http.Response {
	return fgpr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (fgpr FilesystemGetPropertiesResponse) StatusCode() int {
	return fgpr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (fgpr FilesystemGetPropertiesResponse) Status() string {
	return fgpr.rawResponse.Status
}

// Date returns the value for header Date.
func (fgpr FilesystemGetPropertiesResponse) Date() string {
	return fgpr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (fgpr FilesystemGetPropertiesResponse) ETag() string {
	return fgpr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (fgpr FilesystemGetPropertiesResponse) LastModified() string {
	return fgpr.rawResponse.Header.Get("Last-Modified")
}

// XMsNamespaceEnabled returns the value for header x-ms-namespace-enabled.
func (fgpr FilesystemGetPropertiesResponse) XMsNamespaceEnabled() string {
	return fgpr.rawResponse.Header.Get("x-ms-namespace-enabled")
}

// XMsProperties returns the value for header x-ms-properties.
func (fgpr FilesystemGetPropertiesResponse) XMsProperties() string {
	return fgpr.rawResponse.Header.Get("x-ms-properties")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (fgpr FilesystemGetPropertiesResponse) XMsRequestID() string {
	return fgpr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (fgpr FilesystemGetPropertiesResponse) XMsVersion() string {
	return fgpr.rawResponse.Header.Get("x-ms-version")
}

// FilesystemList ...
type FilesystemList struct {
	rawResponse *http.Response
	Filesystems []Filesystem `json:"filesystems,omitempty"`
}

// Response returns the raw HTTP response object.
func (fl FilesystemList) Response() *http.Response {
	return fl.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (fl FilesystemList) StatusCode() int {
	return fl.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (fl FilesystemList) Status() string {
	return fl.rawResponse.Status
}

// ContentType returns the value for header Content-Type.
func (fl FilesystemList) ContentType() string {
	return fl.rawResponse.Header.Get("Content-Type")
}

// Date returns the value for header Date.
func (fl FilesystemList) Date() string {
	return fl.rawResponse.Header.Get("Date")
}

// XMsContinuation returns the value for header x-ms-continuation.
func (fl FilesystemList) XMsContinuation() string {
	return fl.rawResponse.Header.Get("x-ms-continuation")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (fl FilesystemList) XMsRequestID() string {
	return fl.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (fl FilesystemList) XMsVersion() string {
	return fl.rawResponse.Header.Get("x-ms-version")
}

// FilesystemSetPropertiesResponse ...
type FilesystemSetPropertiesResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (fspr FilesystemSetPropertiesResponse) Response() *http.Response {
	return fspr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (fspr FilesystemSetPropertiesResponse) StatusCode() int {
	return fspr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (fspr FilesystemSetPropertiesResponse) Status() string {
	return fspr.rawResponse.Status
}

// Date returns the value for header Date.
func (fspr FilesystemSetPropertiesResponse) Date() string {
	return fspr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (fspr FilesystemSetPropertiesResponse) ETag() string {
	return fspr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (fspr FilesystemSetPropertiesResponse) LastModified() string {
	return fspr.rawResponse.Header.Get("Last-Modified")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (fspr FilesystemSetPropertiesResponse) XMsRequestID() string {
	return fspr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (fspr FilesystemSetPropertiesResponse) XMsVersion() string {
	return fspr.rawResponse.Header.Get("x-ms-version")
}

// Path ...
type Path struct {
	Name *string `json:"name,omitempty"`

	// begin manual edit to generated code
	IsDirectory *bool `json:"isDirectory,string,omitempty"`
	// end manual edit

	LastModified *string `json:"lastModified,omitempty"`
	ETag         *string `json:"eTag,omitempty"`

	// begin manual edit to generated code
	ContentLength *int64 `json:"contentLength,string,omitempty"`
	// end manual edit

	// begin manual addition to generated code
	// TODO:
	//    (a) How can we verify this will actually work with the JSON that the service will emit, when the service starts to do so?
	//    (b) One day, consider converting this to use a custom type, that implements TextMarshaller, as has been done
	//        for the XML-based responses in other SDKs.  For now, the decoding from Base64 is up to the caller, and the name is chosen
	//        to reflect that.
	ContentMD5Base64 *string `json:"contentMd5,string,omitempty"`
	// end manual addition

	Owner       *string `json:"owner,omitempty"`
	Group       *string `json:"group,omitempty"`
	Permissions *string `json:"permissions,omitempty"`
}

// PathCreateResponse ...
type PathCreateResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (pcr PathCreateResponse) Response() *http.Response {
	return pcr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (pcr PathCreateResponse) StatusCode() int {
	return pcr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (pcr PathCreateResponse) Status() string {
	return pcr.rawResponse.Status
}

// ContentLength returns the value for header Content-Length.
func (pcr PathCreateResponse) ContentLength() int64 {
	s := pcr.rawResponse.Header.Get("Content-Length")
	if s == "" {
		return -1
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		i = 0
	}
	return i
}

// Date returns the value for header Date.
func (pcr PathCreateResponse) Date() string {
	return pcr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (pcr PathCreateResponse) ETag() string {
	return pcr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (pcr PathCreateResponse) LastModified() string {
	return pcr.rawResponse.Header.Get("Last-Modified")
}

// XMsContinuation returns the value for header x-ms-continuation.
func (pcr PathCreateResponse) XMsContinuation() string {
	return pcr.rawResponse.Header.Get("x-ms-continuation")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (pcr PathCreateResponse) XMsRequestID() string {
	return pcr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (pcr PathCreateResponse) XMsVersion() string {
	return pcr.rawResponse.Header.Get("x-ms-version")
}

// PathDeleteResponse ...
type PathDeleteResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (pdr PathDeleteResponse) Response() *http.Response {
	return pdr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (pdr PathDeleteResponse) StatusCode() int {
	return pdr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (pdr PathDeleteResponse) Status() string {
	return pdr.rawResponse.Status
}

// Date returns the value for header Date.
func (pdr PathDeleteResponse) Date() string {
	return pdr.rawResponse.Header.Get("Date")
}

// XMsContinuation returns the value for header x-ms-continuation.
func (pdr PathDeleteResponse) XMsContinuation() string {
	return pdr.rawResponse.Header.Get("x-ms-continuation")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (pdr PathDeleteResponse) XMsRequestID() string {
	return pdr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (pdr PathDeleteResponse) XMsVersion() string {
	return pdr.rawResponse.Header.Get("x-ms-version")
}

// PathGetPropertiesResponse ...
type PathGetPropertiesResponse struct {
	rawResponse *http.Response
}

// NewMetadata returns user-defined key/value pairs.
func (bgpr PathGetPropertiesResponse) NewMetadata() (Metadata, error) {
	return NewMetadata(bgpr.XMsProperties())
}

// Response returns the raw HTTP response object.
func (pgpr PathGetPropertiesResponse) Response() *http.Response {
	return pgpr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (pgpr PathGetPropertiesResponse) StatusCode() int {
	return pgpr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (pgpr PathGetPropertiesResponse) Status() string {
	return pgpr.rawResponse.Status
}

// AcceptRanges returns the value for header Accept-Ranges.
func (pgpr PathGetPropertiesResponse) AcceptRanges() string {
	return pgpr.rawResponse.Header.Get("Accept-Ranges")
}

// CacheControl returns the value for header Cache-Control.
func (pgpr PathGetPropertiesResponse) CacheControl() string {
	return pgpr.rawResponse.Header.Get("Cache-Control")
}

// ContentDisposition returns the value for header Content-Disposition.
func (pgpr PathGetPropertiesResponse) ContentDisposition() string {
	return pgpr.rawResponse.Header.Get("Content-Disposition")
}

// ContentEncoding returns the value for header Content-Encoding.
func (pgpr PathGetPropertiesResponse) ContentEncoding() string {
	return pgpr.rawResponse.Header.Get("Content-Encoding")
}

// ContentLanguage returns the value for header Content-Language.
func (pgpr PathGetPropertiesResponse) ContentLanguage() string {
	return pgpr.rawResponse.Header.Get("Content-Language")
}

// ContentLength returns the value for header Content-Length.
func (pgpr PathGetPropertiesResponse) ContentLength() int64 {
	s := pgpr.rawResponse.Header.Get("Content-Length")
	if s == "" {
		return -1
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		i = 0
	}
	return i
}

// ContentMD5 returns the value for header Content-MD5.
// begin manual edit to generated code
func (pgpr PathGetPropertiesResponse) ContentMD5() []byte {
	// TODO: why did I have to generate this myself, whereas for blob API corresponding function seems to be
	//       auto-generated from the Swagger?
	s := pgpr.rawResponse.Header.Get("Content-MD5")
	if s == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		b = nil
	}
	return b
}

// end manual edit to generated code

// ContentRange returns the value for header Content-Range.
func (pgpr PathGetPropertiesResponse) ContentRange() string {
	return pgpr.rawResponse.Header.Get("Content-Range")
}

// ContentType returns the value for header Content-Type.
func (pgpr PathGetPropertiesResponse) ContentType() string {
	return pgpr.rawResponse.Header.Get("Content-Type")
}

// Date returns the value for header Date.
func (pgpr PathGetPropertiesResponse) Date() string {
	return pgpr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (pgpr PathGetPropertiesResponse) ETag() string {
	return pgpr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (pgpr PathGetPropertiesResponse) LastModified() string {
	return pgpr.rawResponse.Header.Get("Last-Modified")
}

// XMsACL returns the value for header x-ms-acl.
func (pgpr PathGetPropertiesResponse) XMsACL() string {
	return pgpr.rawResponse.Header.Get("x-ms-acl")
}

// XMsGroup returns the value for header x-ms-group.
func (pgpr PathGetPropertiesResponse) XMsGroup() string {
	return pgpr.rawResponse.Header.Get("x-ms-group")
}

// XMsLeaseDuration returns the value for header x-ms-lease-duration.
func (pgpr PathGetPropertiesResponse) XMsLeaseDuration() string {
	return pgpr.rawResponse.Header.Get("x-ms-lease-duration")
}

// XMsLeaseState returns the value for header x-ms-lease-state.
func (pgpr PathGetPropertiesResponse) XMsLeaseState() string {
	return pgpr.rawResponse.Header.Get("x-ms-lease-state")
}

// XMsLeaseStatus returns the value for header x-ms-lease-status.
func (pgpr PathGetPropertiesResponse) XMsLeaseStatus() string {
	return pgpr.rawResponse.Header.Get("x-ms-lease-status")
}

// XMsOwner returns the value for header x-ms-owner.
func (pgpr PathGetPropertiesResponse) XMsOwner() string {
	return pgpr.rawResponse.Header.Get("x-ms-owner")
}

// XMsPermissions returns the value for header x-ms-permissions.
func (pgpr PathGetPropertiesResponse) XMsPermissions() string {
	return pgpr.rawResponse.Header.Get("x-ms-permissions")
}

// XMsProperties returns the value for header x-ms-properties.
func (pgpr PathGetPropertiesResponse) XMsProperties() string {
	return pgpr.rawResponse.Header.Get("x-ms-properties")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (pgpr PathGetPropertiesResponse) XMsRequestID() string {
	return pgpr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsResourceType returns the value for header x-ms-resource-type.
func (pgpr PathGetPropertiesResponse) XMsResourceType() string {
	return pgpr.rawResponse.Header.Get("x-ms-resource-type")
}

// XMsVersion returns the value for header x-ms-version.
func (pgpr PathGetPropertiesResponse) XMsVersion() string {
	return pgpr.rawResponse.Header.Get("x-ms-version")
}

// PathLeaseResponse ...
type PathLeaseResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (plr PathLeaseResponse) Response() *http.Response {
	return plr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (plr PathLeaseResponse) StatusCode() int {
	return plr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (plr PathLeaseResponse) Status() string {
	return plr.rawResponse.Status
}

// Date returns the value for header Date.
func (plr PathLeaseResponse) Date() string {
	return plr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (plr PathLeaseResponse) ETag() string {
	return plr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (plr PathLeaseResponse) LastModified() string {
	return plr.rawResponse.Header.Get("Last-Modified")
}

// XMsLeaseID returns the value for header x-ms-lease-id.
func (plr PathLeaseResponse) XMsLeaseID() string {
	return plr.rawResponse.Header.Get("x-ms-lease-id")
}

// XMsLeaseTime returns the value for header x-ms-lease-time.
func (plr PathLeaseResponse) XMsLeaseTime() string {
	return plr.rawResponse.Header.Get("x-ms-lease-time")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (plr PathLeaseResponse) XMsRequestID() string {
	return plr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (plr PathLeaseResponse) XMsVersion() string {
	return plr.rawResponse.Header.Get("x-ms-version")
}

// PathList ...
type PathList struct {
	rawResponse *http.Response
	Paths       []Path `json:"paths,omitempty"`
}

// Response returns the raw HTTP response object.
func (pl PathList) Response() *http.Response {
	return pl.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (pl PathList) StatusCode() int {
	return pl.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (pl PathList) Status() string {
	return pl.rawResponse.Status
}

// Date returns the value for header Date.
func (pl PathList) Date() string {
	return pl.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (pl PathList) ETag() string {
	return pl.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (pl PathList) LastModified() string {
	return pl.rawResponse.Header.Get("Last-Modified")
}

// XMsContinuation returns the value for header x-ms-continuation.
func (pl PathList) XMsContinuation() string {
	return pl.rawResponse.Header.Get("x-ms-continuation")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (pl PathList) XMsRequestID() string {
	return pl.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (pl PathList) XMsVersion() string {
	return pl.rawResponse.Header.Get("x-ms-version")
}

// PathUpdateResponse ...
type PathUpdateResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (pur PathUpdateResponse) Response() *http.Response {
	return pur.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (pur PathUpdateResponse) StatusCode() int {
	return pur.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (pur PathUpdateResponse) Status() string {
	return pur.rawResponse.Status
}

// AcceptRanges returns the value for header Accept-Ranges.
func (pur PathUpdateResponse) AcceptRanges() string {
	return pur.rawResponse.Header.Get("Accept-Ranges")
}

// CacheControl returns the value for header Cache-Control.
func (pur PathUpdateResponse) CacheControl() string {
	return pur.rawResponse.Header.Get("Cache-Control")
}

// ContentDisposition returns the value for header Content-Disposition.
func (pur PathUpdateResponse) ContentDisposition() string {
	return pur.rawResponse.Header.Get("Content-Disposition")
}

// ContentEncoding returns the value for header Content-Encoding.
func (pur PathUpdateResponse) ContentEncoding() string {
	return pur.rawResponse.Header.Get("Content-Encoding")
}

// ContentLanguage returns the value for header Content-Language.
func (pur PathUpdateResponse) ContentLanguage() string {
	return pur.rawResponse.Header.Get("Content-Language")
}

// ContentLength returns the value for header Content-Length.
func (pur PathUpdateResponse) ContentLength() int64 {
	s := pur.rawResponse.Header.Get("Content-Length")
	if s == "" {
		return -1
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		i = 0
	}
	return i
}

// ContentRange returns the value for header Content-Range.
func (pur PathUpdateResponse) ContentRange() string {
	return pur.rawResponse.Header.Get("Content-Range")
}

// ContentType returns the value for header Content-Type.
func (pur PathUpdateResponse) ContentType() string {
	return pur.rawResponse.Header.Get("Content-Type")
}

// Date returns the value for header Date.
func (pur PathUpdateResponse) Date() string {
	return pur.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (pur PathUpdateResponse) ETag() string {
	return pur.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (pur PathUpdateResponse) LastModified() string {
	return pur.rawResponse.Header.Get("Last-Modified")
}

// XMsProperties returns the value for header x-ms-properties.
func (pur PathUpdateResponse) XMsProperties() string {
	return pur.rawResponse.Header.Get("x-ms-properties")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (pur PathUpdateResponse) XMsRequestID() string {
	return pur.rawResponse.Header.Get("x-ms-request-id")
}

// XMsVersion returns the value for header x-ms-version.
func (pur PathUpdateResponse) XMsVersion() string {
	return pur.rawResponse.Header.Get("x-ms-version")
}

// ReadResponse - Wraps the response from the pathClient.Read method.
type ReadResponse struct {
	rawResponse *http.Response
}

// Response returns the raw HTTP response object.
func (rr ReadResponse) Response() *http.Response {
	return rr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (rr ReadResponse) StatusCode() int {
	return rr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (rr ReadResponse) Status() string {
	return rr.rawResponse.Status
}

// Body returns the raw HTTP response object's Body.
func (rr ReadResponse) Body() io.ReadCloser {
	return rr.rawResponse.Body
}

// AcceptRanges returns the value for header Accept-Ranges.
func (rr ReadResponse) AcceptRanges() string {
	return rr.rawResponse.Header.Get("Accept-Ranges")
}

// CacheControl returns the value for header Cache-Control.
func (rr ReadResponse) CacheControl() string {
	return rr.rawResponse.Header.Get("Cache-Control")
}

// ContentDisposition returns the value for header Content-Disposition.
func (rr ReadResponse) ContentDisposition() string {
	return rr.rawResponse.Header.Get("Content-Disposition")
}

// ContentEncoding returns the value for header Content-Encoding.
func (rr ReadResponse) ContentEncoding() string {
	return rr.rawResponse.Header.Get("Content-Encoding")
}

// ContentLanguage returns the value for header Content-Language.
func (rr ReadResponse) ContentLanguage() string {
	return rr.rawResponse.Header.Get("Content-Language")
}

// ContentLength returns the value for header Content-Length.
func (rr ReadResponse) ContentLength() int64 {
	s := rr.rawResponse.Header.Get("Content-Length")
	if s == "" {
		return -1
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		i = 0
	}
	return i
}

// ContentMD5 returns the value for header Content-MD5.
func (rr ReadResponse) ContentMD5() string {
	return rr.rawResponse.Header.Get("Content-MD5")
}

// ContentRange returns the value for header Content-Range.
func (rr ReadResponse) ContentRange() string {
	return rr.rawResponse.Header.Get("Content-Range")
}

// ContentType returns the value for header Content-Type.
func (rr ReadResponse) ContentType() string {
	return rr.rawResponse.Header.Get("Content-Type")
}

// Date returns the value for header Date.
func (rr ReadResponse) Date() string {
	return rr.rawResponse.Header.Get("Date")
}

// ETag returns the value for header ETag.
func (rr ReadResponse) ETag() string {
	return rr.rawResponse.Header.Get("ETag")
}

// LastModified returns the value for header Last-Modified.
func (rr ReadResponse) LastModified() string {
	return rr.rawResponse.Header.Get("Last-Modified")
}

// XMsLeaseDuration returns the value for header x-ms-lease-duration.
func (rr ReadResponse) XMsLeaseDuration() string {
	return rr.rawResponse.Header.Get("x-ms-lease-duration")
}

// XMsLeaseState returns the value for header x-ms-lease-state.
func (rr ReadResponse) XMsLeaseState() string {
	return rr.rawResponse.Header.Get("x-ms-lease-state")
}

// XMsLeaseStatus returns the value for header x-ms-lease-status.
func (rr ReadResponse) XMsLeaseStatus() string {
	return rr.rawResponse.Header.Get("x-ms-lease-status")
}

// XMsProperties returns the value for header x-ms-properties.
func (rr ReadResponse) XMsProperties() string {
	return rr.rawResponse.Header.Get("x-ms-properties")
}

// XMsRequestID returns the value for header x-ms-request-id.
func (rr ReadResponse) XMsRequestID() string {
	return rr.rawResponse.Header.Get("x-ms-request-id")
}

// XMsResourceType returns the value for header x-ms-resource-type.
func (rr ReadResponse) XMsResourceType() string {
	return rr.rawResponse.Header.Get("x-ms-resource-type")
}

// XMsVersion returns the value for header x-ms-version.
func (rr ReadResponse) XMsVersion() string {
	return rr.rawResponse.Header.Get("x-ms-version")
}
