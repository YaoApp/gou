package ssl

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {
	gou.RegisterProcessHandler("ssl.Sign", ProcessSign)
	gou.RegisterProcessHandler("ssl.Verify", ProcessVerify)
}

// ProcessSign computes a signature for the specified data by generating a cryptographic digital signature
func ProcessSign(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	data := process.ArgsString(0)
	certName := process.ArgsString(1)
	algorithm := process.ArgsString(2)

	cert, has := Certificates[certName]
	if !has {
		exception.New("cert %s does not load  ", 400, certName).Throw()
	}

	sign, err := SignStrBase64(data, cert, algorithm)
	if err != nil {
		exception.New("%s", 500, err).Throw()
	}

	return sign
}

// ProcessVerify verifies that the signature is correct for the specified data
func ProcessVerify(process *gou.Process) interface{} {
	process.ValidateArgNums(4)
	data := process.ArgsString(0)
	sign := process.ArgsString(1)
	certName := process.ArgsString(2)
	algorithm := process.ArgsString(3)

	cert, has := Certificates[certName]
	if !has {
		exception.New("cert %s does not load", 400, certName).Throw()
	}

	res, err := VerifyStrBase64(data, sign, cert, algorithm)
	if err != nil {
		exception.New("%s", 500, err).Throw()
	}

	return res
}
