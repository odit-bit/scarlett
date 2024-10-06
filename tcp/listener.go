package tcp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"time"
)

// var defaultConfig = tls.Config{}
// var DefaultTLSConfig = generateTLSConfig()

func DefaultListener(addr string, conf *tls.Config) (net.Listener, error) {
	listener, err := tls.Listen("tcp", addr, conf)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// expose for testing
func DefaultTLSConfig() *tls.Config {
	conf := tls.Config{}
	certBlok, keyBlock, err := GenerateSelfSignedCert(pkix.Name{}, 10*time.Minute, 2048)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.X509KeyPair(certBlok, keyBlock)
	if err != nil {
		log.Fatal(err)
	}
	conf.Certificates = []tls.Certificate{cert}

	return &conf
}

func GenerateSelfSignedCert(subject pkix.Name, validFor time.Duration, keySize int) ([]byte, []byte, error) {
	// generate a new private key
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, err
	}

	// generate a new certificate
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validFor),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	// Add IP SAN to the certificate
	//template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	// encode the certificate and private key
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certPEM, keyPEM, nil
}
