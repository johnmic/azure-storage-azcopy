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
	"fmt"
	"net/url"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/aymanjarrousms/azure-storage-file-go/azfile"
	"github.com/johnmic/azure-storage-azcopy/v10/common/parallel"
)

// Enumerates an entire files account, looking into each matching share as it goes
type fileAccountTraverser struct {
	accountURL    azfile.ServiceURL
	p             pipeline.Pipeline
	ctx           context.Context
	sharePattern  string
	cachedShares  []string
	getProperties bool

	// a generic function to notify that a new stored object has been enumerated
	incrementEnumerationCounter enumerationCounterFunc

	isSync bool

	// For sync operation this flag tells whether this is source or target.
	isSource bool

	// Hierarchical map of files and folders seen on source side.
	indexerMap *folderIndexer

	// child-after-parent ordered communication channel between source and destination traverser.
	orderedTqueue parallel.OrderedTqueueInterface

	// see cookedSyncCmdArgs.maxObjectIndexerSizeInGB for details.
	maxObjectIndexerSizeInGB uint32
}

func (t *fileAccountTraverser) IsDirectory(isSource bool) bool {
	return true // Returns true as account traversal is inherently folder-oriented and recursive.
}

func (t *fileAccountTraverser) listContainers() ([]string, error) {
	if len(t.cachedShares) == 0 {
		marker := azfile.Marker{}
		shareList := make([]string, 0)

		for marker.NotDone() {
			resp, err := t.accountURL.ListSharesSegment(t.ctx, marker, azfile.ListSharesOptions{})

			if err != nil {
				return nil, err
			}

			for _, v := range resp.ShareItems {
				// Match a pattern for the share name and the share name only
				if t.sharePattern != "" {
					if ok, err := containerNameMatchesPattern(v.Name, t.sharePattern); err != nil {
						// Break if the pattern is invalid
						return nil, err
					} else if !ok {
						// Ignore the share if it doesn't match the pattern.
						continue
					}
				}

				shareList = append(shareList, v.Name)
			}

			marker = resp.NextMarker
		}

		t.cachedShares = shareList
		return shareList, nil
	} else {
		return t.cachedShares, nil
	}
}

func (t *fileAccountTraverser) Traverse(preprocessor objectMorpher, processor objectProcessor, filters []ObjectFilter) error {
	// listContainers will return the cached share list if shares have already been listed by this traverser.
	shareList, err := t.listContainers()

	if err != nil {
		return err
	}

	for _, v := range shareList {
		shareURL := t.accountURL.NewShareURL(v).URL()
		shareTraverser := newFileTraverser(&shareURL, t.p, t.ctx, true, t.getProperties, t.incrementEnumerationCounter, t.isSync, t.isSource, t.indexerMap, t.orderedTqueue, t.maxObjectIndexerSizeInGB)

		preprocessorForThisChild := preprocessor.FollowedBy(newContainerDecorator(v))

		err = shareTraverser.Traverse(preprocessorForThisChild, processor, filters)

		if err != nil {
			WarnStdoutAndScanningLog(fmt.Sprintf("failed to list files in share %s: %s", v, err))
			continue
		}
	}

	return nil
}

func newFileAccountTraverser(rawURL *url.URL, p pipeline.Pipeline, ctx context.Context, getProperties bool, incrementEnumerationCounter enumerationCounterFunc, isSync bool, isSource bool, indexerMap *folderIndexer, orderedTqueue parallel.OrderedTqueueInterface, maxObjectIndexerSizeInGB uint32) (t *fileAccountTraverser) {
	fURLparts := azfile.NewFileURLParts(*rawURL)
	sPattern := fURLparts.ShareName

	if fURLparts.ShareName != "" {
		fURLparts.ShareName = ""
	}

	t = &fileAccountTraverser{p: p, ctx: ctx, incrementEnumerationCounter: incrementEnumerationCounter, accountURL: azfile.NewServiceURL(fURLparts.URL(), p), sharePattern: sPattern, getProperties: getProperties,
		isSync:                   isSync,
		isSource:                 isSource,
		orderedTqueue:            orderedTqueue,
		indexerMap:               indexerMap,
		maxObjectIndexerSizeInGB: maxObjectIndexerSizeInGB}
	return
}
