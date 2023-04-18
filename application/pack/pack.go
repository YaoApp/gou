package pack

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/yaoapp/kun/log"
)

var ignorePatterns = map[string]bool{"/data": true, "/db": true, "/logs": true}

// New creates a new package
func New(root string) (*Pack, error) {
	var err error
	if !filepath.IsAbs(root) {
		root, err = filepath.Abs(root)
		if err != nil {
			return nil, err
		}
	}
	return &Pack{PublicKey: nil, Root: root}, nil
}

// Build a package
func (pack *Pack) Build() (string, error) {

	tarPath, err := pack.compress()
	if err != nil {
		return "", err
	}
	defer os.Remove(tarPath)

	encodedPath, err := pack.encode(tarPath)
	if err != nil {
		return "", err
	}

	return encodedPath, nil
}

// BuildTo Build a package
func (pack *Pack) BuildTo(outfile string) error {

	encodedPath, err := pack.Build()
	if err != nil {
		return err
	}

	if _, err := os.Stat(outfile); !os.IsNotExist(err) {
		return fmt.Errorf("file %s already exists", outfile)
	}

	dir := filepath.Dir(outfile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	err = os.Rename(encodedPath, outfile)
	if err != nil {
		return err
	}

	return nil
}

// WithPublicKey takes a public key
func (pack *Pack) WithPublicKey(data []byte) *Pack {

	block, _ := pem.Decode(data)
	if block == nil {
		log.Error("failed to decode PEM block containing public key")
		return pack
	}

	var publicKey *rsa.PublicKey

	switch block.Type {

	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Error("failed to parse public key: %s %s", block.Type, err.Error())
			return pack
		}
		publicKey = cert.PublicKey.(*rsa.PublicKey)
		break

	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			log.Error("failed to parse public key:%s %s", block.Type, err.Error())
			return pack
		}

		if key, ok := key.(*rsa.PublicKey); ok {
			publicKey = key
		}

	default:
		log.Error("unknown key type %s", block.Type)
		return pack
	}

	pack.PublicKey = publicKey
	return pack
}

// WithPublicKeyString takes a public key
func (pack *Pack) WithPublicKeyString(data string) *Pack {
	return pack.WithPublicKey([]byte(data))
}

// WithPublicKeyFile takes a public key file
func (pack *Pack) WithPublicKeyFile(file string) *Pack {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error("failed to parse DER encoded public key: %s", err.Error())
		return pack
	}
	return pack.WithPublicKey(bytes)
}

// WithPrivateKey takes a private key
func (pack *Pack) WithPrivateKey(data []byte) *Pack {

	block, _ := pem.Decode(data)
	if block == nil {
		log.Error("failed to decode PEM block containing public key")
		return pack
	}

	switch block.Type {

	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			log.Error("failed to parse private key: %s %s", block.Type, err.Error())
			return pack
		}

		if key, ok := key.(*rsa.PrivateKey); ok {
			pack.PrivateKey = key
		}
		return pack

	default:
		log.Error("unknown key type %s", block.Type)
		return pack
	}
}

// WithPrivateKeyString takes a private key
func (pack *Pack) WithPrivateKeyString(data string) *Pack {
	return pack.WithPrivateKey([]byte(data))
}

// WithPrivateKeyFile takes a private key file
func (pack *Pack) WithPrivateKeyFile(file string) *Pack {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error("failed to parse DER encoded private key: %s", err.Error())
		return pack
	}
	return pack.WithPrivateKey(bytes)
}

// compress compresses the package
func (pack *Pack) compress() (string, error) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		return "", err
	}

	tarPath := filepath.Join(tempDir, "pack.tar.gz")
	tarFile, err := os.Create(tarPath)
	if err != nil {
		panic(err)
	}
	defer tarFile.Close()

	gz := gzip.NewWriter(tarFile)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	err = filepath.Walk(pack.Root, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(pack.Root, path)
		if err != nil {
			return err
		}

		if ignorePatterns[relPath] {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return tarPath, nil
}

// encoded encodes the package
func (pack *Pack) encode(file string) (string, error) {

	if pack.PublicKey == nil {
		return "", fmt.Errorf("public key is required")
	}

	input, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer input.Close()

	outfile := filepath.Join(filepath.Dir(file), "pack.pkg")
	output, err := os.Create(outfile)
	if err != nil {
		return "", err
	}
	defer output.Close()

	chunkSize := 4096

	// Generate a random symmetric key and encrypt it with the recipient's public key
	symKey := make([]byte, 32)
	if _, err := rand.Read(symKey); err != nil {
		return "", err
	}

	encryptedKey, err := rsa.EncryptPKCS1v15(rand.Reader, pack.PublicKey, symKey)
	if err != nil {
		return "", err
	}

	// Write the encrypted symmetric key to the output file
	if _, err := output.Write(encryptedKey); err != nil {
		return "", err
	}

	// Encrypt the file in chunks using the symmetric key
	block, err := aes.NewCipher(symKey)
	if err != nil {
		return "", err
	}

	buffer := make([]byte, chunkSize)
	cipherWriter := &cipher.StreamWriter{S: cipher.NewCTR(block, make([]byte, aes.BlockSize)), W: output}

	for {
		n, err := input.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}

		if n == 0 {
			break
		}

		if _, err := cipherWriter.Write(buffer[:n]); err != nil {
			return "", err
		}
	}

	if err := cipherWriter.Close(); err != nil {
		return "", err
	}

	return outfile, nil
}

// decode decodes the package
func (pack *Pack) decode(file string) (string, error) {
	// Read the encrypted symmetric key from the input file
	input, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer input.Close()

	encryptedKey := make([]byte, 256)
	if _, err := io.ReadFull(input, encryptedKey); err != nil {
		return "", err
	}

	// Decrypt the symmetric key using the recipient's private key
	symKey, err := rsa.DecryptPKCS1v15(rand.Reader, pack.PrivateKey, encryptedKey)
	if err != nil {
		return "", err
	}

	// Decrypt the file in chunks using the symmetric key
	name := filepath.Base(file)
	outfile := filepath.Join(filepath.Dir(file), name+".tar.gz")
	output, err := os.Create(outfile)
	if err != nil {
		return "", err
	}
	defer output.Close()

	block, err := aes.NewCipher(symKey)
	if err != nil {
		return "", err
	}

	buffer := make([]byte, 4096)
	cipherReader := &cipher.StreamReader{S: cipher.NewCTR(block, make([]byte, aes.BlockSize)), R: input}

	for {
		n, err := cipherReader.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}

		if n == 0 {
			break
		}

		if _, err := output.Write(buffer[:n]); err != nil {
			return "", err
		}
	}

	return outfile, nil
}
