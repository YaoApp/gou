package ssl

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessSign(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := path.Join(root, "certs", "private.pem")
	_, err := LoadCertificateFrom(file, "private")
	if err != nil {
		t.Fatal(err)
	}

	args := []interface{}{"hello world", "private", "SHA256"}
	signature, err := gou.NewProcess("ssl.Sign", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w==", signature)
}

func TestProcessVerify(t *testing.T) {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	file := path.Join(root, "certs", "cert.pem")
	_, err := LoadCertificateFrom(file, "cert")
	if err != nil {
		t.Fatal(err)
	}

	signature := "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w=="
	args := []interface{}{"hello world", signature, "cert", "SHA256"}
	res, err := gou.NewProcess("ssl.Verify", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, res.(bool))
}
