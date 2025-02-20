// Copyright © 2017 Microsoft <wastore@microsoft.com>
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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aymanjarrousms/azure-storage-file-go/azfile"
	"github.com/johnmic/azure-storage-azcopy/v10/azbfs"
	"github.com/johnmic/azure-storage-azcopy/v10/common"
	"github.com/johnmic/azure-storage-azcopy/v10/ste"
)

// extract the right info from cooked arguments and instantiate a generic copy transfer processor from it
func newSyncTransferProcessor(cca *cookedSyncCmdArgs, numOfTransfersPerPart int, fpo common.FolderPropertyOption) *copyTransferProcessor {
	copyJobTemplate := &common.CopyJobPartOrderRequest{
		JobID:           cca.jobID,
		CommandString:   cca.commandString,
		FromTo:          cca.fromTo,
		Fpo:             fpo,
		SourceRoot:      cca.Source.CloneWithConsolidatedSeparators(),
		DestinationRoot: cca.Destination.CloneWithConsolidatedSeparators(),
		CredentialInfo:  cca.credentialInfo,

		// flags
		BlobAttributes: common.BlobTransferAttributes{
			PreserveLastModifiedTime: cca.preserveSMBInfo, // true by default for sync so that future syncs have this information available
			PutMd5:                   cca.putMd5,
			MD5ValidationOption:      cca.md5ValidationOption,
			BlockSizeInBytes:         cca.blockSize},
		ForceWrite:                     common.EOverwriteOption.True(), // once we decide to transfer for a sync operation, we overwrite the destination regardless
		ForceIfReadOnly:                cca.forceIfReadOnly,
		LogLevel:                       AzcopyLogVerbosity,
		PreserveSMBPermissions:         cca.preservePermissions,
		PreserveSMBInfo:                cca.preserveSMBInfo,
		PreservePOSIXProperties:        cca.preservePOSIXProperties,
		S2SSourceChangeValidation:      true,
		DestLengthValidation:           true,
		S2SGetPropertiesInBackend:      true,
		S2SInvalidMetadataHandleOption: common.EInvalidMetadataHandleOption.RenameIfInvalid(),
		CpkOptions:                     cca.cpkOptions,
		S2SPreserveBlobTags:            cca.s2sPreserveBlobTags,

		S2SSourceCredentialType: cca.s2sSourceCredentialType,
	}

	reportFirstPart := func(jobStarted bool) { cca.setFirstPartOrdered() } // for compatibility with the way sync has always worked, we don't check jobStarted here
	reportFinalPart := func() { cca.isEnumerationComplete = true }

	// note that the source and destination, along with the template are given to the generic processor's constructor
	// this means that given an object with a relative path, this processor already knows how to schedule the right kind of transfers
	return newCopyTransferProcessor(copyJobTemplate, numOfTransfersPerPart, cca.Source, cca.Destination,
		reportFirstPart, reportFinalPart, cca.preserveAccessTier, cca.dryrunMode)
}

// base for delete processors targeting different resources
type interactiveDeleteProcessor struct {
	// the plugged-in deleter that performs the actual deletion
	deleter objectProcessor

	// whether we should ask the user for permission the first time we delete a file
	shouldPromptUser bool

	// note down whether any delete should happen
	shouldDelete bool

	// used for prompt message
	// examples: "blob", "local file", etc.
	objectTypeToDisplay string

	// used for prompt message
	// examples: a directory path, or url to container
	objectLocationToDisplay string

	// count the deletions that happened
	incrementDeletionCount func()

	// dryrunMode
	dryrunMode bool
}

func newDeleteTransfer(object StoredObject) (newDeleteTransfer common.CopyTransfer) {
	return common.CopyTransfer{
		Source:             object.relativePath,
		EntityType:         object.entityType,
		LastModifiedTime:   object.lastModifiedTime,
		SourceSize:         object.size,
		ContentType:        object.contentType,
		ContentEncoding:    object.contentEncoding,
		ContentDisposition: object.contentDisposition,
		ContentLanguage:    object.contentLanguage,
		CacheControl:       object.cacheControl,
		Metadata:           object.Metadata,
		BlobType:           object.blobType,
		BlobVersionID:      object.blobVersionID,
		BlobTags:           object.blobTags,
	}
}

func (d *interactiveDeleteProcessor) removeImmediately(object StoredObject) (err error) {
	if d.shouldPromptUser {
		d.shouldDelete, d.shouldPromptUser = d.promptForConfirmation(object) // note down the user's decision
	}

	if !d.shouldDelete {
		return nil
	}

	if d.dryrunMode {
		glcm.Dryrun(func(format common.OutputFormat) string {
			if format == common.EOutputFormat.Json() {
				jsonOutput, err := json.Marshal(newDeleteTransfer(object))
				common.PanicIfErr(err)
				return string(jsonOutput)
			} else { // remove for sync
				if d.objectTypeToDisplay == "local file" { // removing from local src
					dryrunValue := fmt.Sprintf("DRYRUN: remove %v", common.ToShortPath(d.objectLocationToDisplay))
					if runtime.GOOS == "windows" {
						dryrunValue += "\\" + strings.ReplaceAll(object.relativePath, "/", "\\")
					} else { // linux and mac
						dryrunValue += "/" + object.relativePath
					}
					return dryrunValue
				}
				return fmt.Sprintf("DRYRUN: remove %v/%v",
					d.objectLocationToDisplay,
					object.relativePath)
			}
		})
		return nil
	}

	err = d.deleter(object)
	if err != nil {
		glcm.Info(fmt.Sprintf("error %s deleting the object %s", err.Error(), object.relativePath))
	}

	if d.incrementDeletionCount != nil {
		d.incrementDeletionCount()
	}
	return nil // Missing a file is an error, but it's not show-stopping. We logged it earlier; that's OK.
}

func (d *interactiveDeleteProcessor) promptForConfirmation(object StoredObject) (shouldDelete bool, keepPrompting bool) {
	answer := glcm.Prompt(fmt.Sprintf("The %s '%s' does not exist at the source. "+
		"Do you wish to delete it from the destination(%s)?",
		d.objectTypeToDisplay, object.relativePath, d.objectLocationToDisplay),
		common.PromptDetails{
			PromptType:   common.EPromptType.DeleteDestination(),
			PromptTarget: object.relativePath,
			ResponseOptions: []common.ResponseOption{
				common.EResponseOption.Yes(),
				common.EResponseOption.No(),
				common.EResponseOption.YesForAll(),
				common.EResponseOption.NoForAll()},
		},
	)

	switch answer {
	case common.EResponseOption.Yes():
		// print nothing, since the deleter is expected to log the message when the delete happens
		return true, true
	case common.EResponseOption.YesForAll():
		glcm.Info(fmt.Sprintf("Confirmed. All the extra %ss will be deleted.", d.objectTypeToDisplay))
		return true, false
	case common.EResponseOption.No():
		glcm.Info(fmt.Sprintf("Keeping extra %s: %s", d.objectTypeToDisplay, object.relativePath))
		return false, true
	case common.EResponseOption.NoForAll():
		glcm.Info("No deletions will happen from now onwards.")
		return false, false
	default:
		glcm.Info(fmt.Sprintf("Unrecognizable answer, keeping extra %s: %s.", d.objectTypeToDisplay, object.relativePath))
		return false, true
	}
}

func newInteractiveDeleteProcessor(deleter objectProcessor, deleteDestination common.DeleteDestination,
	objectTypeToDisplay string, objectLocationToDisplay common.ResourceString, incrementDeletionCounter func(), dryrun bool) *interactiveDeleteProcessor {

	return &interactiveDeleteProcessor{
		deleter:                 deleter,
		objectTypeToDisplay:     objectTypeToDisplay,
		objectLocationToDisplay: objectLocationToDisplay.Value,
		incrementDeletionCount:  incrementDeletionCounter,
		shouldPromptUser:        deleteDestination == common.EDeleteDestination.Prompt(),
		shouldDelete:            deleteDestination == common.EDeleteDestination.True(), // if shouldPromptUser is true, this will start as false, but we will determine its value later
		dryrunMode:              dryrun,
	}
}

func newSyncLocalDeleteProcessor(cca *cookedSyncCmdArgs) *interactiveDeleteProcessor {
	localDeleter := localFileDeleter{rootPath: cca.Destination.ValueLocal()}
	return newInteractiveDeleteProcessor(localDeleter.deleteFile, cca.deleteDestination, "local file", cca.Destination, cca.incrementDeletionCount, cca.dryrunMode)
}

type localFileDeleter struct {
	rootPath string
}

// As at version 10.4.0, we intentionally don't delete directories in sync,
// even if our folder properties option suggests we should.
// Why? The key difficulties are as follows, and its the third one that we don't currently have a solution for.
//  1. Timing (solvable in theory with FolderDeletionManager)
//  2. Identifying which should be removed when source does not have concept of folders (e.g. BLob)
//     Probably solution is to just respect the folder properties option setting (which we already do in our delete processors)
//  3. In Azure Files case (and to a lesser extent on local disks) users may have ACLS or other properties
//     set on the directories, and wish to retain those even tho the directories are empty. (Perhaps less of an issue
//     when syncing from folder-aware sources that DOES NOT HAVE the directory. But still an issue when syncing from
//     blob. E.g. we delete a folder because there's nothing in it right now, but really user wanted it there,
//     and have set up custom ACLs on it for future use.  If we delete, they lose the custom ACL setup.
//
// TODO: shall we add folder deletion support at some stage? (In cases where folderPropertiesOption says that folders should be processed)
func shouldSyncRemoveFolders() bool {
	return false
}

func (l *localFileDeleter) deleteFile(object StoredObject) error {
	if object.entityType == common.EEntityType.File() {
		glcm.Info("Deleting extra file: " + object.relativePath)
		return os.Remove(common.GenerateFullPath(l.rootPath, object.relativePath))
	}
	if shouldSyncRemoveFolders() {
		panic("folder deletion enabled but not implemented")
	}
	return nil
}

func newSyncDeleteProcessor(cca *cookedSyncCmdArgs, stopDeleteWorkers chan struct{}) (*interactiveDeleteProcessor, error) {
	rawURL, err := cca.Destination.FullURL()
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.TODO(), ste.ServiceAPIVersionOverride, ste.DefaultServiceApiVersion)

	p, err := InitPipeline(ctx, cca.fromTo.To(), cca.credentialInfo, AzcopyLogVerbosity.ToPipelineLogLevel())
	if err != nil {
		return nil, err
	}

	return newInteractiveDeleteProcessor(newRemoteResourceDeleter(rawURL, p, ctx, cca.fromTo.To(), stopDeleteWorkers, cca.incrementDeletionCount, cca.forceIfReadOnly).delete,
		cca.deleteDestination, cca.fromTo.To().String(), cca.Destination, cca.incrementDeletionCount, cca.dryrunMode), nil
}

type remoteResourceDeleter struct {
	rootURL                  *url.URL
	p                        pipeline.Pipeline
	ctx                      context.Context
	targetLocation           common.Location
	blobDeleteChan           chan interface{}
	enumerationDone          chan struct{}
	deleteDirEnumerationChan chan StoredObject
	incrementDeletionCount   func()
	forceIfReadOnly          bool
}

func newRemoteResourceDeleter(rawRootURL *url.URL, p pipeline.Pipeline, ctx context.Context, targetLocation common.Location, enumDone chan struct{}, incrementDeletionCount func(), forceIfReadOnly bool) *remoteResourceDeleter {
	remote := &remoteResourceDeleter{
		rootURL:         rawRootURL,
		p:               p,
		ctx:             ctx,
		targetLocation:  targetLocation,
		forceIfReadOnly: forceIfReadOnly,
		// This channel keep the blobURL to be deleted.
		blobDeleteChan:           make(chan interface{}, 1000*1000),
		deleteDirEnumerationChan: make(chan StoredObject, 1000),

		// Function for incrementing count of files deleted under this folder.
		incrementDeletionCount: incrementDeletionCount,

		// This channel to inform the enumeration complete, workers can stop.
		enumerationDone: enumDone,
	}
	go remote.startDeleteWorkers(ctx)
	return remote
}

// deleteWorker reads the entry from deleteChan and deletes them.
func (b *remoteResourceDeleter) deleteWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case blobURL, ok := <-b.blobDeleteChan:
			if !ok {
				// Delete workers exit when blobDeleteChan is closed by startDeleteWorkers(), which it does when it receives the "exit signal" on the enumerationDone channel,
				// queued by finalize() after target traversal is done.
				return
			}
			// TODO: we are not checking error as of now. We can enqueue those error to errChan.
			//       errChan can be plunbed to error channel for sync. Like we done for copy to know errors at time of enumeration.
			_, err := blobURL.(azblob.BlobURL).Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
			if err != nil {
				fmt.Printf("Blob[%s] deletion failed with error: %v\n", blobURL.(azblob.BlobURL).String(), err)
			} else {
				// One more file deleted inside the folder, update the counter.
				if b.incrementDeletionCount != nil {
					b.incrementDeletionCount()
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// deleteWorker reads the entry from deleteChan and deletes them.
func (b *remoteResourceDeleter) deleteDirEnumerationWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case so, ok := <-b.deleteDirEnumerationChan:
			if !ok {
				// DeleteDirEnumeration workers exit when deleteDirEnumChan is closed by startDeleteWorkers(), which it does when it receives the "exit signal" on the enumerationDone channel,
				// queued by finalize() after target traversal is done.
				return
			}
			b.deleteFolderRecursively(so)
		case <-ctx.Done():
			return
		}
	}
}

const numDeleteWorkers = 64
const numDeleteDirEnumWorkers = 4

// startDeleteWorkers start workers and wait for them to finish.
func (b *remoteResourceDeleter) startDeleteWorkers(ctx context.Context) {
	// wait group for deleteWorkers.
	var wg sync.WaitGroup

	// Wait group for directory enumeration workers.
	var dirEnumWg sync.WaitGroup

	for i := 0; i < numDeleteDirEnumWorkers; i++ {
		dirEnumWg.Add(1)
		go b.deleteDirEnumerationWorker(ctx, &dirEnumWg)
	}

	for i := 0; i < numDeleteWorkers; i++ {
		wg.Add(1)
		go b.deleteWorker(ctx, &wg)
	}

	// Wait for "exit signal" from finalize(), which will be queued when target traverser is done.
	select {
	case <-b.enumerationDone:
		close(b.deleteDirEnumerationChan)
	case <-ctx.Done():
		break
	}

	fmt.Printf("Waiting for deleteDirEnumeration workers to finish\n")
	dirEnumWg.Wait()
	fmt.Printf("deleteDirEnumeration workers done\n")
	// Flag "all directory enumeration workers exited" to delete workers. So that they process all queued files and exit.
	close(b.blobDeleteChan)

	fmt.Printf("Waiting for deleteWorkers to return\n")
	wg.Wait()
	fmt.Printf("deleteWorkers Done\n")

	// Flag "all deleted workers exited" to finalize(). It'll proceed on receiving this signal.
	close(b.enumerationDone)
}

func (b *remoteResourceDeleter) deleteFolderRecursively(object StoredObject) error {
	// sanity check on dirPath.
	if object.relativePath == "" {
		err := fmt.Errorf("cmd::deleteFolder called with empty directory path.")
		panic(err.Error())
	}

	switch b.targetLocation {
	case common.ELocation.Blob():
		// list the folder files and add them to deleteChan for deletion.
		blobURLParts := azblob.NewBlobURLParts(*b.rootURL)
		containerRawURL := copyHandlerUtil{}.getContainerUrl(blobURLParts)
		containerURL := azblob.NewContainerURL(containerRawURL, b.p)

		for marker := (azblob.Marker{}); marker.NotDone(); {
			resp, err := containerURL.ListBlobsHierarchySegment(context.TODO(), marker, "", azblob.ListBlobsSegmentOptions{Prefix: object.relativePath, Details: azblob.BlobListingDetails{
				Metadata: false,
				Deleted:  false,
			}})

			if err != nil {
				fmt.Printf("Folder [%s] Delete failed with error: %v", object.relativePath, err)
				return err
			}

			for _, blobInfo := range resp.Segment.BlobItems {
				// Deleting the blobs underneath the folder, once there is no blob. Folder will be deleted automatically.
				blobURL := containerURL.NewBlobURL(blobInfo.Name)
				b.blobDeleteChan <- blobURL
			}
			marker = resp.NextMarker
		}
		return nil
	case common.ELocation.File():
		err := b.enumerateFileDirectoryDeletion(object) // recursivly delete the sub folder items
		if err != nil {
			fmt.Printf("Object [%s] Deletion failed with error: %v", object.relativePath, err)
			return err
		}
	case common.ELocation.BlobFS():
		blobUrlParts := azbfs.NewBfsURLParts(*b.rootURL)
		blobUrlParts.DirectoryOrFilePath = path.Join(blobUrlParts.DirectoryOrFilePath, object.relativePath)
		dirUrl := azbfs.NewDirectoryURL(blobUrlParts.URL(), b.p)
		marker := ""
		for {
			// When deleting a directory, the number of paths that are deleted with each invocation is limited.
			// If the number of paths to be deleted exceeds this limit, a continuation token is returned in this response header.
			// When a continuation token is returned in the response,
			// it must be specified in a subsequent invocation of the delete operation to continue deleting the directory.
			resp, err := dirUrl.Delete(b.ctx, &marker, true)
			if err != nil {
				fmt.Printf("Folder [%s] Delete failed with error: %v", object.relativePath, err)
				return err
			}

			// update the continuation token for the next call
			marker = resp.XMsContinuation()

			// determine whether listing should be done
			if marker == "" {
				break
			}
		}
	}

	return nil
}

func (b *remoteResourceDeleter) delete(object StoredObject) error {
	if object.entityType == common.EEntityType.File() {
		// TODO: use b.targetLocation.String() in the next line, instead of "object", if we can make it come out as string
		glcm.Info("Deleting extra object: " + object.relativePath)

		var err error
		switch b.targetLocation {
		case common.ELocation.Blob():
			blobURLParts := azblob.NewBlobURLParts(*b.rootURL)
			blobURLParts.BlobName = path.Join(blobURLParts.BlobName, object.relativePath)
			blobURL := azblob.NewBlobURL(blobURLParts.URL(), b.p)
			_, err = blobURL.Delete(b.ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
		case common.ELocation.File():
			fileURLParts := azfile.NewFileURLParts(*b.rootURL)
			fileURLParts.DirectoryOrFilePath = path.Join(fileURLParts.DirectoryOrFilePath, object.relativePath)

			fileURL := azfile.NewFileURL(fileURLParts.URL(), b.p)

			_, err = fileURL.Delete(b.ctx)

			if stgErr, ok := err.(azfile.StorageError); b.forceIfReadOnly && ok && stgErr.ServiceCode() == azfile.ServiceCodeReadOnlyAttribute {
				msg := fmt.Sprintf("read-only attribute detected, removing it before deleting the file %s", object.relativePath)
				if azcopyScanningLogger != nil {
					azcopyScanningLogger.Log(pipeline.LogInfo, msg)
				}

				// if the file is read-only, we need to remove the read-only attribute before we can delete it
				noAttrib := azfile.FileAttributeNone
				_, err = fileURL.SetHTTPHeaders(b.ctx, azfile.FileHTTPHeaders{SMBProperties: azfile.SMBProperties{FileAttributes: &noAttrib}})
				if err == nil {
					_, err = fileURL.Delete(b.ctx)
				} else {
					msg := fmt.Sprintf("error %s removing the read-only attribute from the file %s", err.Error(), object.relativePath)
					glcm.Info(msg + "; check the scanning log file for more details")
					if azcopyScanningLogger != nil {
						azcopyScanningLogger.Log(pipeline.LogError, msg+": "+err.Error())
					}
				}
			}
		case common.ELocation.BlobFS():
			fileURLParts := azbfs.NewBfsURLParts(*b.rootURL)
			fileURLParts.DirectoryOrFilePath = path.Join(fileURLParts.DirectoryOrFilePath, object.relativePath)
			fileURL := azbfs.NewFileURL(fileURLParts.URL(), b.p)
			_, err = fileURL.Delete(b.ctx)
		default:
			panic("not implemented, check your code")
		}

		if err != nil {
			msg := fmt.Sprintf("error %s deleting the object %s", err.Error(), object.relativePath)
			glcm.Info(msg + "; check the scanning log file for more details")
			if azcopyScanningLogger != nil {
				azcopyScanningLogger.Log(pipeline.LogError, msg+": "+err.Error())
			}
		}

		return nil
	} else {
		switch b.targetLocation {
		case common.ELocation.Blob():
			//
			// Let's delete the metaData blob representing the folder inline and rest of folder contents
			// asynchronously by delete workers. This is done to avoid a race condition in the scenario where,
			// since last sync, this folder is deleted and a new file is created with same name.
			// If we defer the metaData blob deletion to the async workers, the deletion may get scheduled after
			// file creation, which will yield undesired results.
			// To avoid this situation, we delete the metaData blob inline so that the file creation can later
			// happen w/o any issues. Rest of the files under the folder can be safely deleted asynchronously as
			// no file/dir with those names need to be created again.
			//
			blobURLParts := azblob.NewBlobURLParts(*b.rootURL)
			blobURLParts.BlobName = path.Join(blobURLParts.BlobName, object.relativePath)
			blobURL := azblob.NewBlobURL(blobURLParts.URL(), b.p)
			_, err := blobURL.Delete(b.ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
			if err != nil {
				if stgErr, ok := err.(azblob.StorageError); ok {
					if stgErr.ServiceCode() != azblob.ServiceCodeBlobNotFound {
						fmt.Printf("Deletion of metablob[%s] representing folder failed with error: %v", object.relativePath, err)
					}
				} else {
					fmt.Printf("Deletion of metablob[%s] representing folder failed with error: %v", object.relativePath, err)
				}
			}

			//
			// Schedule recursive deletion of files/folders under this folder. blobURLParts.BlobName represents
			// meta blob for folder and BlobName contains the absolute path of folder.
			// This is the prefix we should use to list files and directories underneath this folder.
			//
			dirPath := blobURLParts.BlobName + "/"
			b.deleteDirEnumerationChan <- StoredObject{
				relativePath: dirPath,
			}
			return nil

		case common.ELocation.File():
			b.deleteDirEnumerationChan <- object
			return nil
		case common.ELocation.BlobFS():
			b.deleteDirEnumerationChan <- object
			return nil
		default:
			if shouldSyncRemoveFolders() {
				panic("folder deletion enabled but not implemented")
			}
			return nil
		}
	}
}

func (b *remoteResourceDeleter) enumerateFileDirectoryDeletion(object StoredObject) error {
	fileUrlParts := azfile.NewFileURLParts(*b.rootURL)
	fileUrlParts.DirectoryOrFilePath = path.Join(fileUrlParts.DirectoryOrFilePath, object.relativePath)
	dirUrl := azfile.NewDirectoryURL(fileUrlParts.URL(), b.p)

	for marker := (azfile.Marker{}); marker.NotDone(); {
		lResp, err := dirUrl.ListFilesAndDirectoriesSegment(b.ctx, marker, azfile.ListFilesAndDirectoriesOptions{})
		if err != nil {
			fmt.Printf("List folder [%s] failed with error: %v", object.relativePath, err)
		}

		for _, fileInfo := range lResp.FileItems {
			fileURLParts := azfile.NewFileURLParts(*b.rootURL)
			fileURLParts.DirectoryOrFilePath = path.Join(fileURLParts.DirectoryOrFilePath, object.relativePath, fileInfo.Name)
			fileURL := azfile.NewFileURL(fileURLParts.URL(), b.p)

			_, err := fileURL.Delete(b.ctx)

			if stgErr, ok := err.(azfile.StorageError); b.forceIfReadOnly && ok && stgErr.ServiceCode() == azfile.ServiceCodeReadOnlyAttribute {
				msg := fmt.Sprintf("read-only attribute detected, removing it before deleting the file %s", object.relativePath)
				if azcopyScanningLogger != nil {
					azcopyScanningLogger.Log(pipeline.LogInfo, msg)
				}

				// if the file is read-only, we need to remove the read-only attribute before we can delete it
				noAttrib := azfile.FileAttributeNone
				_, err = fileURL.SetHTTPHeaders(b.ctx, azfile.FileHTTPHeaders{SMBProperties: azfile.SMBProperties{FileAttributes: &noAttrib}})
				if err == nil {
					_, err = fileURL.Delete(b.ctx)
				} else {
					msg := fmt.Sprintf("error %s removing the read-only attribute from the file %s", err.Error(), object.relativePath)
					glcm.Info(msg + "; check the scanning log file for more details")
					if azcopyScanningLogger != nil {
						azcopyScanningLogger.Log(pipeline.LogError, msg+": "+err.Error())
					}
				}
			}
			if err != nil {
				// TODO: we are not checking error as of now. We can enqueue those error to errChan.
				// errChan can be plunbed to error channel for sync. Like we done for copy to know errors at time of enumeration.
				fmt.Printf("Deletion of file [%s] failed with error: %v", object.relativePath, err)
			} else if b.incrementDeletionCount != nil {
				b.incrementDeletionCount()
			}
		}
		for _, dirInfo := range lResp.DirectoryItems {
			so := StoredObject{
				relativePath: object.relativePath + "/" + dirInfo.Name,
				entityType:   common.EEntityType.Folder(),
			}

			err := b.enumerateFileDirectoryDeletion(so)
			if err == nil && b.incrementDeletionCount != nil {
				b.incrementDeletionCount()
			}
		}

		marker = lResp.NextMarker
	}
	_, err := dirUrl.Delete(b.ctx)

	if stgErr, ok := err.(azfile.StorageError); b.forceIfReadOnly && ok && stgErr.ServiceCode() == azfile.ServiceCodeReadOnlyAttribute {
		msg := fmt.Sprintf("read-only attribute detected, removing it before deleting the file %s", object.relativePath)
		if azcopyScanningLogger != nil {
			azcopyScanningLogger.Log(pipeline.LogInfo, msg)
		}

		// if the file is read-only, we need to remove the read-only attribute before we can delete it
		noAttrib := azfile.FileAttributeNone
		_, err = dirUrl.SetProperties(b.ctx, azfile.SMBProperties{FileAttributes: &noAttrib})
		if err == nil {
			_, err = dirUrl.Delete(b.ctx)
		} else {
			msg := fmt.Sprintf("error %s removing the read-only attribute from the file %s", err.Error(), object.relativePath)
			glcm.Info(msg + "; check the scanning log file for more details")
			if azcopyScanningLogger != nil {
				azcopyScanningLogger.Log(pipeline.LogError, msg+": "+err.Error())
			}
		}
	}

	if err != nil {
		// TODO: we are not checking error as of now. We can enqueue those error to errChan.
		// errChan can be plunbed to error channel for sync. Like we done for copy to know errors at time of enumeration.
		fmt.Printf("Deletion of folder [%s] failed with error: %v", object.relativePath, err)
	}

	return nil
}
