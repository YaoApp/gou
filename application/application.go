package application

import "github.com/yaoapp/gou/application/disk"

// OpenFromDisk open the application from disk
func OpenFromDisk(root string) (Application, error) {
	return disk.Open(root)
}

// OpenFromPack open the application from the .pkg file
func OpenFromPack(file string) (Application, error) {
	return nil, nil
}

// OpenFromBin open the application from the binary .app file
func OpenFromBin(file string, privateKey string) (Application, error) {
	return nil, nil
}

// OpenFromDB open the application from database
func OpenFromDB(setting interface{}) (Application, error) {
	return nil, nil
}

// OpenFromStore open the application from the store driver
func OpenFromStore(setting interface{}) (Application, error) {
	return nil, nil
}

// OpenFromRemote open the application from the remote source server support .pkg | .app
func OpenFromRemote(url string, auth interface{}) (Application, error) {
	return nil, nil
}

// Parse the json/jsonc/yao/yaml type data
func Parse(data []byte, vPtr interface{}) error {
	return nil
}
