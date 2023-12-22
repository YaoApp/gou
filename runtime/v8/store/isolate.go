package store

import "fmt"

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
