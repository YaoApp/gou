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

// Validate the option
func (option *Option) Validate() {

	if option.MinSize == 0 {
		option.MinSize = 2
	}

	if option.MaxSize == 0 {
		option.MaxSize = 10
	}

	if option.MinSize > 100 {
		log.Warn("[V8] the maximum value of initSize is 100")
		option.MinSize = 100
	}

	if option.MaxSize > 100 {
		log.Warn("[V8] the maximum value of maxSize is 100")
		option.MaxSize = 100
	}

	if option.MinSize > option.MaxSize {
		log.Warn("[V8] the initSize value should smaller than maxSize")
		option.MaxSize = option.MinSize
	}

	if option.HeapSizeLimit == 0 {
		option.HeapSizeLimit = 1518338048 // 1.5G
	}

	if option.HeapSizeLimit > 1518338048 {
		log.Warn("[V8] the maximum value of HeapSizeLimit is 1518338048(1.5G)")
		option.HeapSizeLimit = 1518338048 // 1.5G
	}

	if option.HeapSizeRelease == 0 {
		option.HeapSizeRelease = 52428800 // 50M
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
		option.HeapSizeRelease = 524288000 // 500M
	}
}
