package ssl

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignStrBase64(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("private.pem"), "private")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := SignStrBase64("hello world", cert, "SHA256")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w==", signature)
}

func TestSignHexBase64(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("private.pem"), "private")
	if err != nil {
		t.Fatal(err)
	}
	hexstr := hex.EncodeToString([]byte("hello world"))
	signature, err := SignHexBase64(hexstr, cert, "SHA256")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w==", signature)
}

func TestCertVerifyStrBase64(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("cert.pem"), "cert")
	if err != nil {
		t.Fatal(err)
	}

	signature := "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w=="
	res, err := VerifyStrBase64("hello world", signature, cert, "SHA256")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, res)
}

func TestCertVerifyStrBase64Public(t *testing.T) {
	prepare(t)
	cert, err := Load(certFile("public.pem"), "public")
	if err != nil {
		t.Fatal(err)
	}

	signature := "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w=="
	res, err := VerifyStrBase64("hello world", signature, cert, "SHA256")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, res)
}
