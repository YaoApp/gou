package rest

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// bindHandlers
func (api *API) bindHandlers(router *gin.Engine, option Option) error {
	var group gin.IRoutes = router
	var root = option.Root
	if api.REST.Group != "" && api.REST.Group != "root" {
		paths := append([]string{root}, strings.Split(api.REST.Group, "/")...)
		root = filepath.Join(paths...)
	}

	group = router.Group(root)
	for _, path := range api.REST.Paths {
		err := path.setHandlers(group, api.REST.Guard)
		if err != nil {
			return fmt.Errorf("%s%s%s error: %s", path.Method, root, path.Path, err.Error())
		}
	}
	return nil
}
