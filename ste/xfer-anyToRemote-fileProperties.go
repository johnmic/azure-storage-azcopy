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

package ste

import (
	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/johnmic/azure-storage-azcopy/v10/common"
)

// anyToRemote_folder handles all kinds of sender operations for FOLDERS - both uploads from local files, and S2S copies
func anyToRemote_fileProperties(jptm IJobPartTransferMgr, info TransferInfo, p pipeline.Pipeline, pacer pacer, senderFactory senderFactory, sipf sourceInfoProviderFactory) {
	// schedule the work as a chunk, so it will run on the main goroutine pool, instead of the
	// smaller "transfer initiation pool", where this code runs.
	id := common.NewChunkID(jptm.Info().Source, 0, 0)
	jptm.LogChunkStatus(id, common.EWaitReason.XferStart())

	// step 1. perform initial checks
	if jptm.WasCanceled() {
		/* This is earliest we detect that jptm has been cancelled before we reach destination */
		jptm.SetStatus(common.ETransferStatus.Cancelled())
		jptm.ReportTransferDone()
		return
	}

	// step 2a. Create sender
	srcInfoProvider, err := sipf(jptm)
	if err != nil {
		jptm.LogSendError(info.Source, info.Destination, err.Error(), 0)
		jptm.SetStatus(common.ETransferStatus.Failed())
		jptm.ReportTransferDone()
		return
	}

	if srcInfoProvider.EntityType() != common.EEntityType.FileProperties() {
		panic("configuration error. Source Info Provider does not have FileProperties entity type")
	}

	baseSender, err := senderFactory(jptm, info.Destination, p, pacer, srcInfoProvider)
	if err != nil {
		jptm.LogSendError(info.Source, info.Destination, err.Error(), 0)
		jptm.SetStatus(common.ETransferStatus.Failed())
		jptm.ReportTransferDone()
		return
	}
	s, ok := baseSender.(propertiesSender)
	if !ok {
		jptm.LogSendError(info.Source, info.Destination, "sender implementation does not support copying properties alone", 0)
		jptm.SetStatus(common.ETransferStatus.Failed())
		jptm.ReportTransferDone()
		return
	}

	jptm.LogChunkStatus(id, common.EWaitReason.LockDestination())
	err = jptm.WaitUntilLockDestination(jptm.Context())
	if err != nil {
		jptm.LogSendError(info.Source, info.Destination, err.Error(), 0)
		jptm.SetStatus(common.ETransferStatus.Failed())
		jptm.ReportTransferDone()
		return
	}

	//Just one chunk to schedule
	jptm.SetNumberOfChunks(1)

	// for consistency, always run the standard epilogue
	jptm.SetActionAfterLastChunk(func() { commonSenderCompletion(jptm, baseSender, info) })

	jptm.ScheduleChunks(s.GenerateCopyMetadata(id))
}
