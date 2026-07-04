package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Manager struct {
	mu      sync.Mutex
	dataDir string
	caCert  *x509.Certificate
	caKey   *rsa.PrivateKey
}

type CAInfo struct {
	CertPath    string    `json:"certPath"`
	KeyPath     string    `json:"keyPath"`
	Fingerprint string    `json:"fingerprint"`
	NotAfter    time.Time `json:"notAfter"`
}

func NewManager(dataDir string) (*Manager, error) {
	manager := &Manager{dataDir: filepath.Join(dataDir, "certs")}
	if err := os.MkdirAll(manager.dataDir, 0o700); err != nil {
		return nil, err
	}
	if err := manager.ensureCA(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) Info() (CAInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureCALocked(); err != nil {
		return CAInfo{}, err
	}
	sum := sha256.Sum256(m.caCert.Raw)
	return CAInfo{
		CertPath:    m.caCertPath(),
		KeyPath:     m.caKeyPath(),
		Fingerprint: hex.EncodeToString(sum[:]),
		NotAfter:    m.caCert.NotAfter,
	}, nil
}

func (m *Manager) ServerCertificate(host string) ([]byte, []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureCALocked(); err != nil {
		return nil, nil, err
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, m.caCert, &key.PublicKey, m.caKey)
	if err != nil {
		return nil, nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM, nil
}

func (m *Manager) ensureCA() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureCALocked()
}

func (m *Manager) ensureCALocked() error {
	certPEM, certErr := os.ReadFile(m.caCertPath())
	keyPEM, keyErr := os.ReadFile(m.caKeyPath())
	if certErr == nil && keyErr == nil {
		cert, key, err := parseCA(certPEM, keyPEM)
		if err == nil && cert.NotAfter.After(time.Now().Add(30*24*time.Hour)) {
			m.caCert = cert
			m.caKey = key
			return nil
		}
	}
	cert, key, certBytes, keyBytes, err := generateCA()
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.caCertPath(), certBytes, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(m.caKeyPath(), keyBytes, 0o600); err != nil {
		return err
	}
	m.caCert = cert
	m.caKey = key
	return nil
}

func (m *Manager) caCertPath() string {
	return filepath.Join(m.dataDir, "cursor-assistant-ca.pem")
}

func (m *Manager) caKeyPath() string {
	return filepath.Join(m.dataDir, "cursor-assistant-ca-key.pem")
}

func parseCA(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("missing CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("missing CA private key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func generateCA() (*x509.Certificate, *rsa.PrivateKey, []byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Cursor Assistant Local CA",
			Organization: []string{"Cursor Assistant"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return cert, key, certPEM, keyPEM, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}
