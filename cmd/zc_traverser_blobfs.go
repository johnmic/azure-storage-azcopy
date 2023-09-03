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
package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-pipeline-go/pipeline"

	"github.com/aymanjarrousms/azure-storage-azcopy/v10/azbfs"
	"github.com/aymanjarrousms/azure-storage-azcopy/v10/common"
	"github.com/aymanjarrousms/azure-storage-azcopy/v10/common/parallel"
)

type blobFSTraverser struct {
	rawURL    *url.URL
	p         pipeline.Pipeline
	ctx       context.Context
	recursive bool

	// Generic function to indicate that a new stored object has been enumerated
	incrementEnumerationCounter enumerationCounterFunc

	// Fields applicable only to sync operation.
	// isSync boolean tells whether its copy operation or sync operation.
	isSync bool

	// Error channel for scanning errors
	errorChannel chan ErrorFileInfo

	// For sync operation this flag tells whether this is source or target.
	isSource bool

	// Hierarchical map of files and folders seen on source side.
	indexerMap *folderIndexer

	// child-after-parent ordered communication channel between source and destination traverser.
	orderedTqueue parallel.OrderedTqueueInterface

	// Map for checking rename directories, so that complete sub-tree can be enumerated.
	possiblyRenamedMap *possiblyRenamedMap

	// see cookedSyncCmdArgs.maxObjectIndexerSizeInGB for details.
	maxObjectIndexerSizeInGB uint32

	// see cookedSyncCmdArgs.lastSyncTime for details.
	lastSyncTime time.Time

	// See cookedSyncCmdArgs.cfdMode for details.
	cfdMode common.CFDMode

	// see cookedSyncCmdArgs.metaDataOnlySync for details.
	metaDataOnlySync bool

	// scannerLogger to log scanning error.
	scannerLogger common.ILoggerResetable
}

func newBlobFSTraverser(rawURL *url.URL, p pipeline.Pipeline, ctx context.Context, recursive bool, incrementEnumerationCounter enumerationCounterFunc, isSync, isSource bool,
	errorChannel chan ErrorFileInfo, indexerMap *folderIndexer, possiblyRenamedMap *possiblyRenamedMap, orderedTqueue parallel.OrderedTqueueInterface, maxObjectIndexerSizeInGB uint32,
	lastSyncTime time.Time, cfdMode common.CFDMode, metaDataOnlySync bool, scannerLogger common.ILoggerResetable) (t *blobFSTraverser) {
	t = &blobFSTraverser{
		rawURL:                      rawURL,
		p:                           p,
		ctx:                         ctx,
		recursive:                   recursive,
		incrementEnumerationCounter: incrementEnumerationCounter,
		isSync:                      isSync,
		isSource:                    isSource,
		orderedTqueue:               orderedTqueue,
		possiblyRenamedMap:          possiblyRenamedMap,
		errorChannel:                errorChannel,
		indexerMap:                  indexerMap,
		maxObjectIndexerSizeInGB:    maxObjectIndexerSizeInGB,
		lastSyncTime:                lastSyncTime,
		cfdMode:                     cfdMode,
		metaDataOnlySync:            metaDataOnlySync,
		scannerLogger:               scannerLogger,
	}
	return
}

func (t *blobFSTraverser) IsDirectory(bool) bool {
	return copyHandlerUtil{}.urlIsBFSFileSystemOrDirectory(t.ctx, t.rawURL, t.p) // This gets all the fanciness done for us.
}

func (t *blobFSTraverser) getPropertiesIfSingleFile() (*azbfs.PathGetPropertiesResponse, bool, error) {
	pathURL := azbfs.NewFileURL(*t.rawURL, t.p)
	pgr, err := pathURL.GetProperties(t.ctx)

	if err != nil {
		return nil, false, err
	}

	if pgr.XMsResourceType() == "directory" {
		return pgr, false, nil
	}

	return pgr, true, nil
}

func (_ *blobFSTraverser) parseLMT(t string) time.Time {
	out, err := time.Parse(time.RFC1123, t)

	if err != nil {
		return time.Time{}
	}

	return out
}

func (t *blobFSTraverser) getFolderProps() (p contentPropsProvider, size int64) {
	return noContentProps, 0
}

func (t *blobFSTraverser) writeToErrorChannel(err ErrorFileInfo) {
	if t.scannerLogger != nil {
		t.scannerLogger.Log(pipeline.LogError, err.ErrorMsg.Error())
	} else {
		WarnStdoutAndScanningLog(err.ErrorMsg.Error())
	}
	if t.errorChannel != nil {
		t.errorChannel <- err
	}
}

func (t *blobFSTraverser) parallelSyncTargetEnumeration(directoryURL azbfs.DirectoryURL, enumerateOneDir parallel.EnumerateOneDirFunc, processor objectProcessor, filters []ObjectFilter) error {
	// initiate parallel scanning, starting at the root path
	workerContext, cancelWorkers := context.WithCancel(t.ctx)
	channels := parallel.Crawl(workerContext, directoryURL, "" /* relBase */, enumerateOneDir, EnumerationParallelism, func() int64 {
		if t.indexerMap != nil {
			return t.indexerMap.getObjectIndexerMapSize()
		}
		panic("ObjectIndexerMap is nil")
	}, t.orderedTqueue, t.isSource, t.isSync, t.maxObjectIndexerSizeInGB)

	errChan := make(chan error, len(channels))
	var wg sync.WaitGroup
	processFunc := func(index int) {
		defer wg.Done()
		for {
			select {
			case x, ok := <-channels[index]:
				if !ok {
					return
				}

				if x.EnqueueToTqueue() {
					// Do the sanity check, EnqueueToTqueue should be true in case of sync operation and traverser is source.
					if !t.isSync || !t.isSource {
						panic(fmt.Sprintf("Entry set for enqueue to tqueue for invalid operation, isSync[%v], isSource[%v]", t.isSync, t.isSource))
					}
					_, err := x.Item()
					if err != nil {
						panic(fmt.Sprintf("Error set for entry which needs to be inserted to tqueue: %v", err))
					}
					//
					// This is a special CrawlResult which signifies that we need to enqueue the given directory to tqueue for
					// target traverser to process. Tell orderedTqueue so that it can add it in a proper child-after-parent
					// order.
					//
					item, _ := x.Item()
					t.orderedTqueue.MarkProcessed(x.Idx(), item)
					continue
				}

				item, workerError := x.Item()
				if workerError != nil {
					errChan <- workerError
					cancelWorkers()
					return
				}

				object := item.(StoredObject)

				if t.incrementEnumerationCounter != nil {
					t.incrementEnumerationCounter(object.entityType)
				}

				processErr := processIfPassedFilters(filters, object, processor)
				_, processErr = getProcessingError(processErr)
				if processErr != nil {
					fmt.Printf("Traverser failed with error: %v", processErr)
					errChan <- processErr
					cancelWorkers()
					return
				}
			case err := <-errChan:
				fmt.Printf("Some other thread received error, so coming out.")
				// Requeue the error for other go routines to read.
				errChan <- err
				return
			}
		}
	}

	for i := 0; i < len(channels); i++ {
		wg.Add(1)
		go processFunc(i)
	}
	wg.Wait()

	fmt.Printf("Done processing of blobfs traverser channels")
	if len(errChan) > 0 {
		err := <-errChan
		return err
	}
	return nil
}

//
// Given a directory find out if it has changed since the last sync. A “changed” directory could mean one or more of the following:
// 1. One or more new files/subdirs created inside the directory.
// 2. One or more files/subdirs deleted.
// 3. Directory is renamed.
//
// It honours CFDMode to make the decision, f.e., if CFDMode allows ctime/mtime to be used for CFD it may not need to query attributes from target.
//
// If it returns True, TargetTraverser will enumerate the directory and compare each enumerated object with the source scanned objects in
// ObjectIndexer[] to find out if the object needs to be sync'ed.
//
// For CFDModes that allow ctime for CFD we can avoid enumerating target dir if we know directory has not changed. This increases sync efficiency.
//
// Note: If we discover that certain sources cannot be safely trusted for ctime update we can change this to return True for them, thus falling back
// on the more rigorous target<->source comparison. //
func (t *blobFSTraverser) hasDirectoryChangedSinceLastSync(currentDirPath string) bool {
	// Get the storedObject for currentDirPath to compare Ctime, Mtime for different mode.
	// Although we don't need for CFDMode == TargetCompare, but need for sanity check.
	so := t.indexerMap.getStoredObject(currentDirPath)

	if currentDirPath != so.relativePath {
		panic(fmt.Sprintf("curentDirPath[%s] not matched with storedObject relative path[%s]", currentDirPath, so.relativePath))
	}

	// Force enumeration for TargetCompare mode. For other CFDModes we enumerate a directory iff it has changed since last sync.
	if t.cfdMode == common.CFDModeFlags.TargetCompare() {
		return true
	}

	//
	// If CFDMode allows using ctime, compare directory ctime with LastSyncTime.
	// Note that directory ctime will change if a new object is created inside the
	// directory or an existing object is deleted or the directory is renamed.
	//
	if t.cfdMode == common.CFDModeFlags.CtimeMtime() {
		if so.lastChangeTime.After(t.lastSyncTime) {
			return true
		}
		return false
	} else if t.cfdMode == common.CFDModeFlags.Ctime() {
		if so.lastChangeTime.After(t.lastSyncTime) {
			return true
		} else if t.metaDataOnlySync && t.indexerMap.filesChangedInDirectory(so.relativePath, t.lastSyncTime) {
			//
			// If metaDataOnlySync is true and a file has "ctime > LastSyncTime" and CFDMode does not allow us to use mtime for checking if the
			// file's data+metadata or only metadata has changed, then we need to compare the file's source attributes with target attributes.
			// Since fetching attributes for individual target file may be expensive for some targets (f.e. Blob), so it would be better to enumerate
			// the target parent dir which will be cheaper due to ListDir returning many files and their attributes in a single call. In normal scenario,
			// there will be less than 2K files in a folder and all 2K files along with their attributes can be retrieved in a single ListDir call.
			// If not even one file changed in a directory we don't need to compare attributes for any file and hence we don't fetch the attributes.
			//
			return true
		} else {
			return false
		}
	} else {
		panic(fmt.Sprintf("Unsupported CFDMode: %d", t.cfdMode))
	}
}

func (t *blobFSTraverser) generateDirUrl(blobFSUrlParts azbfs.BfsURLParts, relativePath string) azbfs.DirectoryURL {
	fileUrl := blobFSUrlParts.URL()
	if relativePath != "" && !strings.HasPrefix(relativePath, common.AZCOPY_PATH_SEPARATOR_STRING) {
		relativePath = common.AZCOPY_PATH_SEPARATOR_STRING + relativePath
	}

	fileUrl.Path = strings.TrimSuffix(fileUrl.Path, common.AZCOPY_PATH_SEPARATOR_STRING) + relativePath
	return azbfs.NewDirectoryURL(fileUrl, t.p)
}

//
// waitTillAllAncestorsAreProcessed will block the calling thread till all
// ancestors of 'currentDirPath' are processed and removed from ObjectIndexerMap.
// f.e., if 'currentDirPath' is "a/b/c/d", then it'll wait till there are any of the
// following directories present in the ObjectIndexerMap.
// "", "a", "a/b", "a/b/c"
//
// This will be called by TargetTraverser to ensure that it processes a
// directory from tqueue only after all its ancestors. The reason being that
// unless we process every ancestor of a directory we cannot safely conclude
// if any of the ancestors could be the result of a rename. Note that if a
// directory has any ancestor that is renamed, it means that the directory
// MUST be enumerated in order to copy its contents and hence we must know
// that when we process the directory from tqueue.
//
func (t *blobFSTraverser) waitTillAllAncestorsAreProcessed(currentDirPath string) {
	//
	// If possiblyRenamedMap is nil then we are not doing the special rename handling, hence
	// directories can be processed in any order (no need to wait for ancestors).
	//
	if t.possiblyRenamedMap == nil {
		return
	}

	if t.cfdMode == common.CFDModeFlags.TargetCompare() {
		panic("waitTillAllAncestorsAreProcessed() MUST NOT be called for TargetCompare mode!")
	}
	//
	// Empty currentDirPath represents root folder, root folder has no ancestors, so no need to wait.
	//
	if currentDirPath == "" {
		return
	}

	//
	// Wait for the parent to be processed by the Target Traverser.
	// As a directory is processed it'll be removed from t.indexerMap.folderMap[], so we know that the directory
	// is processed by waiting till it's no more present in t.indexerMap.folderMap[].
	// Since parent processing would have in turn waited for its parent and so on, we just need to wait for the
	// immediate parent to be sure that all ancestors of currentDirPath are processed by Target Traverser.
	//
	parent := filepath.Dir(currentDirPath)

	//
	// waitCount to track how much time spent waiting for directory to be processed.
	//
	waitCount := 0
	for {
		if t.indexerMap.directoryNotYetProcessed(parent) {
			time.Sleep(10 * time.Millisecond)
			waitCount++
			t.scannerLogger.Log(pipeline.LogInfo, fmt.Sprintf("currentDir(%s) parent(%s) still not processed after iteration[%v]", currentDirPath, parent, waitCount))
		} else {
			// parent directory processed by target traverser, now it can safely proceed with this directory.
			break
		}

		//
		// In most practical case it should be small wait, but to be safe lets use a fairly large wait.
		// If directory still not processed it means some bug.
		//
		if waitCount > 3600*100 {
			panic(fmt.Sprintf("currentDir(%s) parent(%s) still not processed after 3600 secs\n", currentDirPath, parent))
		}
	}

	// Sanity check, If parent of currentDirPath processed from ObjectIndexerMap, GrandParent entry should also be processed.
	if parent != "." {
		grandParent := filepath.Dir(parent)
		if t.indexerMap.directoryNotYetProcessed(grandParent) {
			panic(fmt.Sprintf("We should not never reach here, where for child(%s), parent(%s) entry processed, "+
				"but grandParent(%s) entry still present in ObjectIndexerMap", currentDirPath, parent, grandParent))
		}
	}
}

//
// hasAnAncestorThatIsPossiblyRenamed checks all ancestors of 'currentDirPath' starting
// from the immediate parent and walking upwards till the root and returns
// true if any of the ancestor is present in 'possiblyRenamedMap'.
// Directories are added to 'possiblyRenamedMap' in two places:
// 1. For directories that exist in the target, it's added to
//    'possiblyRenamedMap' if the target directory's inode is
//    different from source directory's inode.
// 2. For directories that don't exist in the target, all of them are added
//    to 'possiblyRenamedMap'. It could very well be a new directory but
//    that doesn't cause any additional overhead since new directories
//    are anyway attempted enumeration.
//
// Note: This must never be called for CFDmode==TargetCompare.
//
func (t *blobFSTraverser) hasAnAncestorThatIsPossiblyRenamed(currentDirPath string) bool {
	if t.cfdMode == common.CFDModeFlags.TargetCompare() {
		panic(fmt.Sprintf("hasAnAncestorThatIsPossiblyRenamed called for currentDirPath(%s) with TargetCompare", currentDirPath))
	}

	// possiblyRenamedMap can be nil. As of now we have possiblyRenamedMap to detect any potential rename.
	// In future, we may use different approach to detect rename as optimization.
	if t.possiblyRenamedMap == nil {
		return false
	}

	//
	// "" signifies root, root does not have any ancestor and hence cannot be renamed.
	//
	if currentDirPath == "" {
		return false
	}

	for ; currentDirPath != "."; currentDirPath = filepath.Dir(currentDirPath) {
		if t.possiblyRenamedMap.exists(currentDirPath) {
			return true
		}
	}
	return false
}

func (t *blobFSTraverser) Traverse(preprocessor objectMorpher, processor objectProcessor, filters []ObjectFilter) (err error) {
	bfsURLParts := azbfs.NewBfsURLParts(*t.rawURL)
	targetTraverser := t.isSync && !t.isSource

	if bfsURLParts.DirectoryOrFilePath != "" {
		pathProperties, isFile, _ := t.getPropertiesIfSingleFile()
		if isFile {
			if azcopyScanningLogger != nil {
				azcopyScanningLogger.Log(pipeline.LogDebug, "Detected the root as a file.")
			}

			storedObject := newStoredObject(
				preprocessor,
				getObjectNameOnly(bfsURLParts.DirectoryOrFilePath),
				"",
				common.EEntityType.File(),
				t.parseLMT(pathProperties.LastModified()),
				pathProperties.ContentLength(),
				md5OnlyAdapter{md5: pathProperties.ContentMD5()}, // not supplying full props, since we can't below, and it would be inconsistent to do so here
				noBlobProps,
				noMetdata, // not supplying metadata, since we can't below and it would be inconsistent to do so here
				bfsURLParts.FileSystemName,
			)

			if t.incrementEnumerationCounter != nil {
				t.incrementEnumerationCounter(common.EEntityType.File())
			}

			err := processIfPassedFilters(filters, storedObject, processor)
			_, err = getProcessingError(err)
			return err
		}
	}

	directoryURL := azbfs.NewDirectoryURL(bfsURLParts.URL(), t.p)

	enumerateOneDir := func(dir parallel.Directory, enqueueDir func(parallel.Directory), enqueueOutput func(parallel.DirectoryEntry, error)) error {
		currentDirPath := dir.(string)
		var currentDirURL azbfs.DirectoryURL
		//
		// Flag to be passed in FinalizeDirectory to indicate to the receiver (processIfNecessary()) if it should copy all the files or call
		// HasFileChangedSinceLastSyncUsingLocalChecks() to find out files that need to be copied.
		//
		FinalizeAll := false
		if targetTraverser {
			//
			// Don't proceed with processing this directory right away, wait till all of its ancestors are processed. We do this by periodically
			// checking ObjectIndexerMap to make sure there are no ancestors left in the ObjectIndexerMap.
			// This is important because as ancestor directories are processed from tqueue it may cause new directories to be added to
			// "possiblyRenamedMap" which we will need to correctly process this directory.
			// For CFDMode == TargetCompare, this should be a no-op.
			//
			t.waitTillAllAncestorsAreProcessed(currentDirPath)

			// If source directory has not changed since last sync, then we don't really need to enumerate the target. SourceTraverser would have enumerated
			// this directory and added all the children in ObjectIndexer map. We just need to go over these objects and find out which of these need to be
			// sync'ed to target and sync them appropriately (data+metadata or only metadata).
			//
			// If directory has changed then there could be some files deleted and to find them we need to enumerate the target directory and compare. Also,
			// if directory is renamed then also it'll be considered changed. Note that a renamed directory needs to be fully enumerated at the target as even
			// files with same names as in the target could be entirely different files. This forces us to enumerate the target directory if the source
			// directory is seen to have changed, since we don’t know if it was renamed, in which case we must enumerate the target directory.
			if !t.hasDirectoryChangedSinceLastSync(currentDirPath) && !t.hasAnAncestorThatIsPossiblyRenamed(currentDirPath) {
				goto FinalizeDirectory
			}

			// in case of target sync, we are getting only relative path, so we should generate the full dir path
			currentDirURL = t.generateDirUrl(bfsURLParts, currentDirPath)
		} else {
			currentDirURL = dir.(azbfs.DirectoryURL)
		}

		//
		// Now that we are going to enumerate the target dir, we will process all files present in the target, and when FinalizeDirectory is called only those
		// files will be left in the ObjectIndexer map which are newly created in source. We need to blindly copy all of these to the target.
		// Set FinalizeAll to true to convey this to the receiver (processIfNecessary()).
		//
		FinalizeAll = true
		for marker := (azbfs.Marker{}); marker.NotDone(); {
			lResp, err := currentDirURL.ListDirectorySegment(t.ctx, marker, false /* check if should be true or false*/)
			if err != nil {
				if targetTraverser && err.(azbfs.StorageError).Response().StatusCode == 404 {
					storedObject := StoredObject{
						name:              getObjectNameOnly(strings.TrimSuffix(dir.(string), common.AZCOPY_PATH_SEPARATOR_STRING)),
						relativePath:      strings.TrimSuffix(dir.(string), common.AZCOPY_PATH_SEPARATOR_STRING),
						entityType:        common.EEntityType.Folder(),
						ContainerName:     bfsURLParts.FileSystemName,
						isFolderEndMarker: true,
						isFinalizeAll:     true,
					}

					enqueueOutput(storedObject, nil)
					return nil
				}

				return fmt.Errorf("cannot list files due to reason %s", err)
			}

			for _, fileInfo := range lResp.Files() {

				fileRelativePath := strings.TrimPrefix(*fileInfo.Name, bfsURLParts.DirectoryOrFilePath)
				fileRelativePath = strings.TrimPrefix(fileRelativePath, common.AZCOPY_PATH_SEPARATOR_STRING)

				subFileURL := directoryURL.NewFileURL(fileRelativePath)
				resp, err := subFileURL.GetProperties(t.ctx)

				if err != nil {
					return fmt.Errorf("cannot get file properties due to reason %s", err)
				}

				metadata, err := resp.NewMetadata()
				if err != nil {
					return fmt.Errorf("cannot get file metadata due to reason %s", err)
				}

				storedObject := StoredObject{
					name:               *fileInfo.Name,
					relativePath:       fileRelativePath,
					entityType:         common.EEntityType.File(),
					lastModifiedTime:   fileInfo.LastModifiedTime(),
					size:               resp.ContentLength(),
					cacheControl:       resp.CacheControl(),
					contentDisposition: resp.ContentDisposition(),
					contentEncoding:    resp.ContentEncoding(),
					contentLanguage:    resp.ContentLanguage(),
					contentType:        resp.ContentType(),
					md5:                resp.ContentMD5(),
					Metadata:           common.FromAzBlobFSMetadataToCommonMetadata(metadata),
					ContainerName:      bfsURLParts.FileSystemName,
				}

				extendedProp, _ := common.ReadStatFromBlobFSMetadata(metadata, resp.ContentLength())
				storedObject.lastChangeTime = extendedProp.CTime()
				storedObject.lastModifiedTime = extendedProp.MTime()

				enqueueOutput(storedObject, nil)
			}

			for _, dirInfo := range lResp.Directories() {

				folderRelativePath := strings.TrimPrefix(dirInfo, bfsURLParts.DirectoryOrFilePath)
				folderRelativePath = strings.TrimPrefix(folderRelativePath, common.AZCOPY_PATH_SEPARATOR_STRING)

				subDirURL := t.generateDirUrl(bfsURLParts, folderRelativePath)
				resp, err := subDirURL.GetProperties(t.ctx)

				metadata, _ := resp.NewMetadata()
				extendedProp, _ := common.ReadStatFromBlobFSMetadata(metadata, resp.ContentLength())
				storedObject := StoredObject{
					name:               folderRelativePath,
					relativePath:       folderRelativePath,
					entityType:         common.EEntityType.Folder(),
					lastModifiedTime:   extendedProp.MTime(),
					size:               resp.ContentLength(),
					cacheControl:       resp.CacheControl(),
					contentDisposition: resp.ContentDisposition(),
					contentEncoding:    resp.ContentEncoding(),
					contentLanguage:    resp.ContentLanguage(),
					contentType:        resp.ContentType(),
					md5:                resp.ContentMD5(),
					Metadata:           common.FromAzBlobFSMetadataToCommonMetadata(metadata),
					ContainerName:      bfsURLParts.FileSystemName,
				}

				storedObject.lastChangeTime = extendedProp.CTime()
				storedObject.lastModifiedTime = extendedProp.MTime()
				storedObject.inode = extendedProp.INode()

				if !targetTraverser {
					enqueueOutput(storedObject, err)
				}

				if t.recursive {
					// If recursive is turned on, add sub directories to be
					// in case of target sync, we will not enqueue this dir using enqueue dir, since the enumerateOneDir should enumerates only folders that exists in the source
					// So we will not enumerate destination directories that was deleted in the source
					if !targetTraverser {
						enqueueDir(currentDirURL.NewDirectoryURL(dirInfo))
					} else {
						// In case of folder that exist only in the destination, the folder should enumerate by the target traveres
						// So that the comparere will be able to know that this directory should be treated
						if err == nil {
							enqueueOutput(storedObject, err)
						}
					}

				}
			}

			// if debug mode is on, note down the result, this is not going to be fast
			if azcopyScanningLogger != nil && azcopyScanningLogger.ShouldLog(pipeline.LogDebug) {
				tokenValue := "NONE"
				if marker.Val != nil {
					tokenValue = *marker.Val
				}

				var vdirListBuilder strings.Builder
				for _, file := range lResp.Files() {
					fmt.Fprintf(&vdirListBuilder, " %s,", *file.Name)
				}
				var fileListBuilder strings.Builder
				for _, directory := range lResp.Directories() {
					fmt.Fprintf(&fileListBuilder, " %s,", directory)
				}
				msg := fmt.Sprintf("Enumerating %s with token %s. Sub-dirs:%s Files:%s", currentDirPath,
					tokenValue, vdirListBuilder.String(), fileListBuilder.String())
				azcopyScanningLogger.Log(pipeline.LogDebug, msg)
			}

			continuationMarker := lResp.XMsContinuation()
			marker = azbfs.Marker{Val: &continuationMarker}
		}
	FinalizeDirectory:
		if targetTraverser {
			// This storedObject marks the end of folder enumeration. Comparator after recieving end marker
			// do the finalize operation on this directory.
			// Note: Kept the containerName for debugging, mainly for multiple job with same source and different container.
			storedObject := StoredObject{
				name:              getObjectNameOnly(strings.TrimSuffix(currentDirPath, common.AZCOPY_PATH_SEPARATOR_STRING)),
				relativePath:      strings.TrimSuffix(currentDirPath, common.AZCOPY_PATH_SEPARATOR_STRING),
				entityType:        common.EEntityType.Folder(),
				ContainerName:     bfsURLParts.FileSystemName,
				isFolderEndMarker: true,
				isFinalizeAll:     FinalizeAll,
			}
			enqueueOutput(storedObject, nil)
		}
		return nil
	}

	if targetTraverser {
		return t.parallelSyncTargetEnumeration(directoryURL, enumerateOneDir, processor, filters)
	}

	return
}

// globalBlobFSMd5ValidationOption is an ugly workaround, to tweak performance of another ugly workaround (namely getContentMd5, below)
var globalBlobFSMd5ValidationOption = common.EHashValidationOption.FailIfDifferentOrMissing() // default to strict, if not set

// getContentMd5 compensates for the fact that ADLS Gen 2 currently does not return MD5s in the PathListResponse (even
// tho the property is there in the swagger and the generated API)
func (t *blobFSTraverser) getContentMd5(ctx context.Context, directoryURL azbfs.DirectoryURL, file azbfs.Path) []byte {
	if globalBlobFSMd5ValidationOption == common.EHashValidationOption.NoCheck() {
		return nil // not gonna check it, so don't need it
	}

	var returnValueForError []byte = nil // If we get an error, we just act like there was no content MD5. If validation is set to fail on error, this will fail the transfer of this file later on (at the time of the MD5 check)

	// convert format of what we have, if we have something in the PathListResponse from Service
	if file.ContentMD5Base64 != nil {
		value, err := base64.StdEncoding.DecodeString(*file.ContentMD5Base64)
		if err != nil {
			return returnValueForError
		}
		return value
	}

	// Fall back to making a new round trip to the server
	// This is an interim measure, so that we can still validate MD5s even before they are being returned in the server's
	// PathList response
	// TODO: remove this in a future release, once we know that Service is always returning the MD5s in the PathListResponse.
	//     Why? Because otherwise, if there's a file with NO MD5, we'll make a round-trip here, but that's pointless if we KNOW that
	//     that Service is always returning them in the PathListResponse which we've already checked above.
	//     As at mid-Feb 2019, we don't KNOW that (in fact it's not returning them in the PathListResponse) so we need this code for now.
	fileURL := directoryURL.FileSystemURL().NewDirectoryURL(*file.Name)
	props, err := fileURL.GetProperties(ctx)
	if err != nil {
		return returnValueForError
	}
	return props.ContentMD5()
}
