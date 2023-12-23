package store

import (
	"fmt"
)

const (

	// IsoReady isolate is ready
	IsoReady uint8 = 0

	// IsoBusy isolate is in used
	IsoBusy uint8 = 1
)

// Dispose the isolate
func (iso *Isolate) Dispose() {
	// fmt.Printf("dispose isolate: %s\n", iso.Key())
	Isolates.Remove(iso.Key()) // remove from normal isolates
	iso.Isolate.Dispose()
	iso.Isolate = nil
	iso.Template = nil
	iso = nil
}

// Key return the key of the isolate
func (iso *Isolate) Key() string {
	return fmt.Sprintf("%p", iso)
}

// Lock the isolate
func (iso *Isolate) Lock() {
	iso.Status = IsoBusy
}

// Unlock the isolate
func (iso *Isolate) Unlock() {
	iso.Status = IsoReady
}

// Locked check if the isolate is locked
func (iso *Isolate) Locked() bool {
	return iso.Status == IsoBusy
}

// Health  check the isolate health
func (iso *Isolate) Health(HeapSizeRelease uint64, HeapAvailableSize uint64) bool {

	// {
	// 	"ExternalMemory": 0,
	// 	"HeapSizeLimit": 1518338048,
	// 	"MallocedMemory": 16484,
	// 	"NumberOfDetachedContexts": 0,
	// 	"NumberOfNativeContexts": 3,
	// 	"PeakMallocedMemory": 24576,
	// 	"TotalAvailableSize": 1518051356,
	// 	"TotalHeapSize": 1261568,
	// 	"TotalHeapSizeExecutable": 262144,
	// 	"TotalPhysicalSize": 499164,
	// 	"UsedHeapSize": 713616
	// }

	if iso.Isolate == nil {
		return false
	}

	stat := iso.Isolate.GetHeapStatistics()
	if stat.TotalAvailableSize < HeapAvailableSize { // 500M
		return false
	}

	return true
}
