// Copyright © Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package e2etest

import (
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aymanjarrousms/azure-storage-file-go/azfile"
	"github.com/johnmic/azure-storage-azcopy/v10/azbfs"
	"github.com/johnmic/azure-storage-azcopy/v10/common"
)

func assertNoStripTopDir(stripTopDir bool) {
	if stripTopDir {
		panic("support for stripTopDir is not yet implemented here") // when implemented, resourceManagers should return /* in the right part of the string
	}
}

type downloadContentOptions struct {
	resourceRelPath string
	downloadBlobContentOptions
	downloadFileContentOptions
	downloadBlobFSContentOptions
}

// nolint
type downloadBlobContentOptions struct {
	containerURL azblob.ContainerURL
	cpkInfo      common.CpkInfo
	cpkScopeInfo common.CpkScopeInfo
}

type downloadBlobFSContentOptions struct {
	fileSystemUrl azbfs.FileSystemURL
}

type downloadFileContentOptions struct {
	shareURL azfile.ShareURL
}

// TODO: any better names for this?
// a source or destination. We need one of these for each of Blob, Azure Files, BlobFS, S3, Local disk etc.
type resourceManager interface {

	// creates an empty container/share/directory etc
	createLocation(a asserter, s *scenario)

	// creates the test files in the location. Implementers can assume that createLocation has been called first.
	// This method may be called multiple times, in which case it should overwrite any like-named files that are already there.
	// (e.g. when test need to create files with later modification dates, they will trigger a second call to this)
	createFiles(a asserter, s *scenario, isSource bool)

	// creates a test file in the location. Same assumptions as createFiles.
	createFile(a asserter, o *testObject, s *scenario, isSource bool)

	// Gets the names and properties of all files (and, if applicable, folders) that exist.
	// Used for verification
	getAllProperties(a asserter) map[string]*objectProperties

	// Download
	downloadContent(a asserter, options downloadContentOptions) []byte

	// cleanup gets rid of everything that setup created
	// (Takes no param, because the resourceManager is expected to track its own state. E.g. "what did I make")
	cleanup(a asserter)

	// gets the azCopy command line param that represents the resource.  withSas is ignored when not applicable
	getParam(stripTopDir bool, withSas bool, withFile string) string

	getSAS() string

	// isContainerLike returns true if the resource is a top-level cloud-based resource (e.g. a container, a File Share, etc)
	isContainerLike() bool

	// appendSourcePath appends a path to creates absolute path
	appendSourcePath(string, bool)

	// create a snapshot for the source, and use it for the job
	createSourceSnapshot(a asserter)
}

// /////////////

type resourceLocal struct {
	dirPath string
	baseDir string
}

func (r *resourceLocal) createLocation(a asserter, s *scenario) {
	if r.dirPath == common.Dev_Null {
		return
	}

	r.baseDir = ""
	if s.fromTo == common.FromTo(ETestFromTo.SMBMountFile()) {
		r.baseDir = "/mnt/AzCopyE2ESMB"
	}

	r.dirPath = TestResourceFactory{}.CreateLocalDirectory(a, r.baseDir)
	if s.GetModifiableParameters().relativeSourcePath != "" {
		r.appendSourcePath(s.GetModifiableParameters().relativeSourcePath, true)
	}
}

func (r *resourceLocal) createFiles(a asserter, s *scenario, isSource bool) {
	if r.dirPath == common.Dev_Null {
		return
	}

	scenarioHelper{}.generateLocalFilesFromList(a, &generateLocalFilesFromList{
		dirPath: r.dirPath,
		generateFromListOptions: generateFromListOptions{
			fs:          s.fs.allObjects(isSource),
			defaultSize: s.fs.defaultSize,
		},
	})
}

func (r *resourceLocal) createFile(a asserter, o *testObject, s *scenario, isSource bool) {
	if r.dirPath == common.Dev_Null {
		return
	}

	scenarioHelper{}.generateLocalFilesFromList(a, &generateLocalFilesFromList{
		dirPath: r.dirPath,
		generateFromListOptions: generateFromListOptions{
			fs:          []*testObject{o},
			defaultSize: s.fs.defaultSize,
		},
	})
}

func (r *resourceLocal) cleanup(_ asserter) {
	if r.dirPath == common.Dev_Null {
		return
	}

	if r.dirPath != "" {
		_ = os.RemoveAll(r.dirPath)
	}
}

func (r *resourceLocal) getParam(stripTopDir bool, withSas bool, withFile string) string {
	if r.dirPath == common.Dev_Null {
		return common.Dev_Null
	}

	if !stripTopDir {
		if withFile != "" {
			p := path.Join(r.dirPath, withFile)

			if runtime.GOOS == "windows" {
				p = strings.ReplaceAll(p, "/", "\\")
			}

			return p
		}

		return r.dirPath
	}
	return path.Join(r.dirPath, "*")
}

func (r *resourceLocal) getSAS() string {
	return ""
}

func (r *resourceLocal) isContainerLike() bool {
	return false
}

func (r *resourceLocal) appendSourcePath(filePath string, _ bool) {
	r.dirPath += "/" + filePath
}

func (r *resourceLocal) getAllProperties(a asserter) map[string]*objectProperties {
	if r.dirPath == common.Dev_Null {
		return make(map[string]*objectProperties)
	}

	return scenarioHelper{}.enumerateLocalProperties(a, r.dirPath)
}

func (r *resourceLocal) downloadContent(_ asserter, _ downloadContentOptions) []byte {
	panic("Not Implemented")
}

func (r *resourceLocal) createSourceSnapshot(a asserter) {
	panic("Not Implemented")
}

// /////

type resourceBlobContainer struct {
	accountType  AccountType
	containerURL *azblob.ContainerURL
	rawSasURL    *url.URL
}

func (r *resourceBlobContainer) createLocation(a asserter, s *scenario) {
	cu, _, rawSasURL := TestResourceFactory{}.CreateNewContainer(a, s.GetTestFiles().sourcePublic, r.accountType)
	r.containerURL = &cu
	r.rawSasURL = &rawSasURL
	if s.GetModifiableParameters().relativeSourcePath != "" {
		r.appendSourcePath(s.GetModifiableParameters().relativeSourcePath, true)
	}
}

func (r *resourceBlobContainer) createFiles(a asserter, s *scenario, isSource bool) {
	options := &generateBlobFromListOptions{
		rawSASURL:    *r.rawSasURL,
		containerURL: *r.containerURL,
		generateFromListOptions: generateFromListOptions{
			fs:          s.fs.allObjects(isSource),
			defaultSize: s.fs.defaultSize,
			accountType: s.srcAccountType,
		},
	}
	if s.fromTo.IsDownload() {
		options.cpkInfo = common.GetCpkInfo(s.p.cpkByValue)
		options.cpkScopeInfo = common.GetCpkScopeInfo(s.p.cpkByName)
	}
	if isSource {
		options.accessTier = s.p.accessTier
	}
	scenarioHelper{}.generateBlobsFromList(a, options)
}

func (r *resourceBlobContainer) createFile(a asserter, o *testObject, s *scenario, isSource bool) {
	options := &generateBlobFromListOptions{
		containerURL: *r.containerURL,
		generateFromListOptions: generateFromListOptions{
			fs:          []*testObject{o},
			defaultSize: s.fs.defaultSize,
		},
	}

	if s.fromTo.IsDownload() {
		options.cpkInfo = common.GetCpkInfo(s.p.cpkByValue)
		options.cpkScopeInfo = common.GetCpkScopeInfo(s.p.cpkByName)
	}

	scenarioHelper{}.generateBlobsFromList(a, options)
}

func (r *resourceBlobContainer) cleanup(a asserter) {
	if r.containerURL != nil {
		deleteContainer(a, *r.containerURL)
	}
}

func (r *resourceBlobContainer) getParam(stripTopDir bool, withSas bool, withFile string) string {
	var uri url.URL
	if withSas {
		uri = *r.rawSasURL
	} else {
		uri = r.containerURL.URL()
	}

	if withFile != "" {
		bURLParts := azblob.NewBlobURLParts(uri)

		bURLParts.BlobName = withFile

		uri = bURLParts.URL()
	}

	return uri.String()
}

func (r *resourceBlobContainer) getSAS() string {
	return "?" + r.rawSasURL.RawQuery
}

func (r *resourceBlobContainer) isContainerLike() bool {
	return true
}

func (r *resourceBlobContainer) appendSourcePath(filePath string, useSas bool) {
	if useSas {
		r.rawSasURL.Path += "/" + filePath
	}
}

func (r *resourceBlobContainer) getAllProperties(a asserter) map[string]*objectProperties {
	return scenarioHelper{}.enumerateContainerBlobProperties(a, *r.containerURL)
}

func (r *resourceBlobContainer) downloadContent(a asserter, options downloadContentOptions) []byte {
	options.containerURL = *r.containerURL
	return scenarioHelper{}.downloadBlobContent(a, options)
}

func (r *resourceBlobContainer) createSourceSnapshot(a asserter) {
	panic("Not Implemented")
}

// ///

type resourceBlobFSContainer struct {
	accountType   AccountType
	fileSystemURL *azbfs.FileSystemURL
	rawSasURL     *url.URL
}

func (r *resourceBlobFSContainer) createLocation(a asserter, s *scenario) {
	fsu, _, rawSasURL := TestResourceFactory{}.CreateNewFileSystem(a, r.accountType)
	r.fileSystemURL = &fsu
	r.rawSasURL = &rawSasURL
	if s.GetModifiableParameters().relativeSourcePath != "" {
		r.appendSourcePath(s.GetModifiableParameters().relativeSourcePath, true)
	}
}

func (r *resourceBlobFSContainer) createFiles(a asserter, s *scenario, isSource bool) {
	options := &generateBlobFSFromListOptions{
		rawSASURL:    *r.rawSasURL,
		containerURL: *r.fileSystemURL,
		generateFromListOptions: generateFromListOptions{
			fs:          s.fs.allObjects(isSource),
			defaultSize: s.fs.defaultSize,
			accountType: s.srcAccountType,
		},
	}
	scenarioHelper{}.generateBlobFSFromList(a, options)
}

func (r *resourceBlobFSContainer) createFile(a asserter, o *testObject, s *scenario, isSource bool) {
	options := &generateBlobFSFromListOptions{
		containerURL: *r.fileSystemURL,
		generateFromListOptions: generateFromListOptions{
			fs:          []*testObject{o},
			defaultSize: s.fs.defaultSize,
		},
	}

	scenarioHelper{}.generateBlobFSFromList(a, options)
}

func (r *resourceBlobFSContainer) cleanup(a asserter) {
	if r.fileSystemURL != nil {
		deleteFilesystem(a, *r.fileSystemURL)
	}
}

func (r *resourceBlobFSContainer) getParam(stripTopDir bool, withSas bool, withFile string) string {
	var uri url.URL
	if withSas {
		uri = *r.rawSasURL
	} else {
		uri = r.fileSystemURL.URL()
	}

	if withFile != "" {
		bfsURLParts := azbfs.NewBfsURLParts(uri)

		bfsURLParts.DirectoryOrFilePath = withFile

		uri = bfsURLParts.URL()
	}

	return uri.String()
}

func (r *resourceBlobFSContainer) getSAS() string {
	return "?" + r.rawSasURL.RawQuery
}

func (r *resourceBlobFSContainer) isContainerLike() bool {
	return true
}

func (r *resourceBlobFSContainer) appendSourcePath(filePath string, useSas bool) {
	if useSas {
		r.rawSasURL.Path += "/" + filePath
	}
}

func (r *resourceBlobFSContainer) getAllProperties(a asserter) map[string]*objectProperties {
	return scenarioHelper{}.enumerateContainerBlobFSProperties(a, *r.fileSystemURL)
}

func (r *resourceBlobFSContainer) downloadContent(a asserter, options downloadContentOptions) []byte {
	options.fileSystemUrl = *r.fileSystemURL
	return scenarioHelper{}.downloadBlobFSContent(a, options)
}

func (r *resourceBlobFSContainer) createSourceSnapshot(a asserter) {
	panic("Not Implemented")
}

// ///

type resourceAzureFileShare struct {
	accountType AccountType
	shareURL    *azfile.ShareURL // // TODO: Either eliminate SDK URLs from ResourceManager or provide means to edit it (File SDK) for which pipeline is required
	rawSasURL   *url.URL
	snapshotID  string // optional, use a snapshot as the location instead
}

func (r *resourceAzureFileShare) createLocation(a asserter, s *scenario) {
	su, _, rawSasURL := TestResourceFactory{}.CreateNewFileShare(a, EAccountType.Standard())
	r.shareURL = &su
	r.rawSasURL = &rawSasURL
	if s.GetModifiableParameters().relativeSourcePath != "" {
		r.appendSourcePath(s.GetModifiableParameters().relativeSourcePath, true)
	}
}

func (r *resourceAzureFileShare) createFiles(a asserter, s *scenario, isSource bool) {
	scenarioHelper{}.generateAzureFilesFromList(a, &generateAzureFilesFromListOptions{
		shareURL:    *r.shareURL,
		fileList:    s.fs.allObjects(isSource),
		defaultSize: s.fs.defaultSize,
	})
}

func (r *resourceAzureFileShare) createFile(a asserter, o *testObject, s *scenario, isSource bool) {
	scenarioHelper{}.generateAzureFilesFromList(a, &generateAzureFilesFromListOptions{
		shareURL:    *r.shareURL,
		fileList:    []*testObject{o},
		defaultSize: s.fs.defaultSize,
	})
}

func (r *resourceAzureFileShare) cleanup(a asserter) {
	if r.shareURL != nil {
		deleteShare(a, *r.shareURL)
	}
}

func (r *resourceAzureFileShare) getParam(stripTopDir bool, withSas bool, withFile string) string {
	assertNoStripTopDir(stripTopDir)
	var param url.URL
	if withSas {
		param = *r.rawSasURL
	} else {
		param = r.shareURL.URL()
	}

	// append the snapshot ID if present
	if r.snapshotID != "" || withFile != "" {
		parts := azfile.NewFileURLParts(param)
		if r.snapshotID != "" {
			parts.ShareSnapshot = r.snapshotID
		}

		if withFile != "" {
			parts.DirectoryOrFilePath = withFile
		}
		param = parts.URL()
	}

	return param.String()
}

func (r *resourceAzureFileShare) getSAS() string {
	return "?" + r.rawSasURL.RawQuery
}

func (r *resourceAzureFileShare) isContainerLike() bool {
	return true
}

func (r *resourceAzureFileShare) appendSourcePath(filePath string, useSas bool) {
	if useSas {
		r.rawSasURL.Path += "/" + filePath
	}
}

func (r *resourceAzureFileShare) getAllProperties(a asserter) map[string]*objectProperties {
	return scenarioHelper{}.enumerateShareFileProperties(a, *r.shareURL)
}

func (r *resourceAzureFileShare) downloadContent(a asserter, options downloadContentOptions) []byte {
	return scenarioHelper{}.downloadFileContent(a, downloadContentOptions{
		resourceRelPath: options.resourceRelPath,
		downloadFileContentOptions: downloadFileContentOptions{
			shareURL: *r.shareURL,
		},
	})
}

func (r *resourceAzureFileShare) createSourceSnapshot(a asserter) {
	r.snapshotID = TestResourceFactory{}.CreateNewFileShareSnapshot(a, *r.shareURL)
}

// //

type resourceDummy struct{}

func (r *resourceDummy) createLocation(a asserter, s *scenario) {

}

func (r *resourceDummy) createFiles(a asserter, s *scenario, isSource bool) {

}

func (r *resourceDummy) createFile(a asserter, o *testObject, s *scenario, isSource bool) {

}

func (r *resourceDummy) cleanup(_ asserter) {
}

func (r *resourceDummy) getParam(stripTopDir bool, withSas bool, withFile string) string {
	assertNoStripTopDir(stripTopDir)
	return ""
}

func (r *resourceDummy) getSAS() string {
	return ""
}

func (r *resourceDummy) isContainerLike() bool {
	return false
}

func (r *resourceDummy) getAllProperties(a asserter) map[string]*objectProperties {
	panic("not impelmented")
}

func (r *resourceDummy) downloadContent(a asserter, options downloadContentOptions) []byte {
	return make([]byte, 0)
}

func (r *resourceDummy) appendSourcePath(_ string, _ bool) {
}

func (r *resourceDummy) createSourceSnapshot(a asserter) {}
