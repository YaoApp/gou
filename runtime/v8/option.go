package v8

import "github.com/yaoapp/kun/log"

// SetHeapAvailableSize set runtime Available
func SetHeapAvailableSize(size uint) {
	runtimeOption.HeapAvailableSize = uint64(size)
}

// DisablePrecompile disable the precompile
func DisablePrecompile() {
	runtimeOption.Precompile = false
}

// EnablePrecompile enable the precompile feature
func EnablePrecompile() {
	runtimeOption.Precompile = true
}

// EnableDebug enable the debug mode
func EnableDebug() {
	runtimeOption.Debug = true
}

// Validate the option
func (option *Option) Validate() {

	if option.MinSize == 0 {
		option.MinSize = 50
	}

	if option.MaxSize == 0 {
		option.MaxSize = 100
	}

	if option.DefaultTimeout == 0 {
		option.DefaultTimeout = 200
	}

	if option.ContextTimeout == 0 {
		option.ContextTimeout = 200
	}

	if option.ContetxQueueSize == 0 {
		option.ContetxQueueSize = 10
	}

	if option.Mode == "" {
		option.Mode = "standard"
	}

	if option.MinSize > 500 {
		log.Warn("[V8] the maximum value of initSize is 500")
		option.MinSize = 500
	}

	if option.MaxSize > 500 {
		log.Warn("[V8] the maximum value of maxSize is 500")
		option.MaxSize = 500
	}

	if option.MinSize > option.MaxSize {
		log.Warn("[V8] the initSize value should smaller than maxSize")
		option.MaxSize = option.MinSize
	}

	if option.HeapSizeLimit == 0 {
		option.HeapSizeLimit = 1518338048 // 1.5G
	}

	if option.HeapSizeLimit > 4294967296 {
		log.Warn("[V8] the maximum value of HeapSizeLimit is 4294967296(4G)")
		option.HeapSizeLimit = 1518338048 // 1.5G
	}

	if option.HeapSizeRelease == 0 {
		option.HeapSizeRelease = 524288 // 50M
	}

	if option.HeapSizeRelease > 524288000 {
		log.Warn("[V8] the maximum value of heapSizeRelease is 524288000(500M)")
		option.HeapSizeRelease = 524288000 // 500M
	}

	if option.HeapAvailableSize == 0 {
		option.HeapAvailableSize = 524288000 // 500M
	}

	if option.HeapAvailableSize < 524288000 || option.HeapAvailableSize > option.HeapSizeLimit {
		log.Warn("[V8] the heapAvailableSize value is 524288000(500M) or heapSizeLimit * 0.30 to reduce the risk of program crashes")
		// option.HeapSizeRelease = 524288000 // 500M
	}
}
