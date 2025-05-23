package certs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/ocelot-cloud/shared/validation"
	"log"
	"math/big"
	"net"
	"net/http"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"sync"
	"time"
)

const (
	TLS_CERT = "TLS_CERT"
	TLS_KEY  = "TLS_KEY"
)

var (
	Logger      = tools.Logger
	currentCert *tls.Certificate
	rwCertMutex sync.RWMutex
)

func StartServers(handler http.Handler) {
	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       10 * time.Minute,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			Logger.Error("HTTP server failed: %v", err)
		}
	}()
	startTlsServer(handler)
}

func startTlsServer(handler http.Handler) {
	isPresent, err := isCertPresent()
	if err != nil {
		Logger.Fatal("Failed to check for certificate: %v", err)
	}
	if isPresent {
		currentCert = loadCertFromDatabase()
	} else {
		currentCert = generateSelfSignedCertAndSaveToDatabase()
	}

	srv := &http.Server{
		Addr:    ":8443",
		Handler: handler,
		TLSConfig: &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: dynamicCertProvider(),
		},
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       10 * time.Minute,
		ErrorLog:          log.New(errorLoggerWhichDropsBadCertMessages{}, "", 0),
	}
	err = srv.ListenAndServeTLS("", "")
	if err != nil {
		Logger.Error("Failed to start tls server: %v", err)
	}
}

type errorLoggerWhichDropsBadCertMessages struct{}

func (w errorLoggerWhichDropsBadCertMessages) Write(p []byte) (int, error) {
	// Although this behavior is expected, when the ocelotcloud container connects to or disconnects from an app network, the server causes a NETWORK_CHANGED error as an HTTP response. This message shall be ignored to avoid confusion.
	if bytes.Contains(p, []byte("TLS handshake error")) &&
		bytes.Contains(p, []byte("unknown certificate")) {
		return len(p), nil // swallow only this class of message
	}
	Logger.Error("TLS handshake error: %s", string(p))
	return len(p), nil
}

func generateSelfSignedCertAndSaveToDatabase() *tls.Certificate {
	cert, err := GenerateUniversalSelfSignedCert()
	if err != nil {
		Logger.Fatal("Failed to generate self-signed certificate: %v", err)
	}
	err = SaveCert(cert)
	if err != nil {
		Logger.Error("Failed to save self-signed certificate: %v", err)
	}
	return cert
}

func loadCertFromDatabase() *tls.Certificate {
	cert, err := loadCert()
	if err != nil {
		Logger.Error("Failed to load certificate: %v", err)
		Logger.Error("Falling back to freshly generated self-signed cert")
		cert, err = GenerateUniversalSelfSignedCert()
		if err != nil {
			Logger.Fatal("Failed to generate self-signed certificate: %v", err)
		}
	}
	return cert
}

func GenerateUniversalSelfSignedCert() (*tls.Certificate, error) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		DNSNames:     []string{"*", "*.*"},
		IPAddresses:  []net.IP{net.IPv4(0, 0, 0, 0), net.IPv6zero},
		IsCA:         false,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	tlsCert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, err
	}

	return &tlsCert, nil
}

func isCertPresent() (bool, error) {
	var tlsCertCount int
	err := common.DB.QueryRow("SELECT COUNT(*) FROM configs WHERE key = $1", TLS_CERT).Scan(&tlsCertCount)
	if err != nil {
		return false, err
	}
	if tlsCertCount != 1 {
		return false, nil
	}

	var tlsKeyCount int
	err = common.DB.QueryRow("SELECT COUNT(*) FROM configs WHERE key = $1", TLS_KEY).Scan(&tlsKeyCount)
	if err != nil {
		return false, err
	}
	if tlsKeyCount != 1 {
		return false, nil
	}
	return true, nil
}

func loadCert() (*tls.Certificate, error) {
	rwCertMutex.RLock()
	defer rwCertMutex.RUnlock()
	var certPEM, keyPEM []byte
	err := common.DB.QueryRow("SELECT value FROM configs WHERE key = $1", TLS_CERT).Scan(&certPEM)
	if err != nil {
		return nil, err
	}

	err = common.DB.QueryRow("SELECT value FROM configs WHERE key = $1", TLS_KEY).Scan(&keyPEM)
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tlsCert, nil
}

func SaveCert(cert *tls.Certificate) error {
	rwCertMutex.Lock()
	defer rwCertMutex.Unlock()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))})

	tx, err := common.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO configs (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2", TLS_CERT, certPEM)
	if err != nil {
		err = tx.Rollback()
		if err != nil {
			Logger.Error("Failed to rollback transaction: %v", err)
			return err
		}
		return err
	}

	_, err = tx.Exec("INSERT INTO configs (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2", TLS_KEY, keyPEM)
	if err != nil {
		err = tx.Rollback()
		if err != nil {
			Logger.Error("Failed to rollback transaction: %v", err)
			return err
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		Logger.Error("Failed to commit transaction: %v", err)
		return err
	}

	currentCert = cert
	return nil
}

func dynamicCertProvider() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		rwCertMutex.RLock()
		defer rwCertMutex.RUnlock()
		if currentCert == nil {
			Logger.Error("No certificate found")
		}
		return currentCert, nil
	}
}

type Blob struct {
	Data []byte `json:"data"`
}

func CertificateUploadHandler(w http.ResponseWriter, r *http.Request) {
	blob, err := validation.ReadBody[Blob](w, r)
	if err != nil {
		return
	}

	var certPEM, keyPEM []byte
	rest := blob.Data
	for {
		block, r := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = r
		switch block.Type {
		case "CERTIFICATE":
			certPEM = append(certPEM, pem.EncodeToMemory(block)...)
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			keyPEM = append(keyPEM, pem.EncodeToMemory(block)...)
		}
	}
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		Logger.Warn("Failed to parse certificate: %v", err)
		http.Error(w, "invalid certificate/key", http.StatusBadRequest)
		return
	}
	if err = SaveCert(&tlsCert); err != nil {
		Logger.Warn("Failed to save certificate: %v", err)
		http.Error(w, "failed to save cert", http.StatusInternalServerError)
		return
	}
	tools.WriteResponse(w, "certificate uploaded successfully")
}

func ConvertToFullchainPemBytes(clientCert *tls.Certificate) ([]byte, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCert.Certificate[0]})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(clientCert.PrivateKey)
	if err != nil {
		tools.Logger.Warn("Failed to marshal private key: %v", err)
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	pemData := append(certPEM, keyPEM...)
	return pemData, nil
}
