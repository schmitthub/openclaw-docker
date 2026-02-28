package render

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func generateCA(opts Options) error {
	certPath := filepath.Join(opts.OutputDir, "ca-cert.pem")
	keyPath := filepath.Join(opts.OutputDir, "ca-key.pem")

	// Preserve existing CA across re-runs so gateway trust isn't broken.
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "OpenClaw Docker Egress CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("create CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if opts.ConfirmWrite != nil {
		if err := opts.ConfirmWrite(certPath); err != nil {
			return err
		}
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("write CA cert %q: %w", certPath, err)
	}

	if opts.ConfirmWrite != nil {
		if err := opts.ConfirmWrite(keyPath); err != nil {
			return err
		}
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write CA key %q: %w", keyPath, err)
	}

	return nil
}

func generateNginxCert(opts Options) error {
	certPath := filepath.Join(opts.OutputDir, "nginx-cert.pem")
	keyPath := filepath.Join(opts.OutputDir, "nginx-key.pem")

	// Preserve existing cert across re-runs.
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	// Load the CA cert+key to sign the server certificate.
	caCertPath := filepath.Join(opts.OutputDir, "ca-cert.pem")
	caKeyPath := filepath.Join(opts.OutputDir, "ca-key.pem")

	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("read CA cert for nginx signing: %w", err)
	}
	caKeyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return fmt.Errorf("read CA key for nginx signing: %w", err)
	}

	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		return fmt.Errorf("decode CA cert PEM: no PEM block found")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA certificate: %w", err)
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil {
		return fmt.Errorf("decode CA key PEM: no PEM block found")
	}
	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA private key: %w", err)
	}

	// Generate a new key pair for the nginx server cert.
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate nginx server key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "openclaw-gateway",
		},
		DNSNames:    []string{"localhost", "openclaw-gateway"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(2 * 365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create nginx server certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return fmt.Errorf("marshal nginx server key: %w", err)
	}
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER})

	if opts.ConfirmWrite != nil {
		if err := opts.ConfirmWrite(certPath); err != nil {
			return err
		}
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("write nginx cert %q: %w", certPath, err)
	}

	if opts.ConfirmWrite != nil {
		if err := opts.ConfirmWrite(keyPath); err != nil {
			return err
		}
	}
	if err := os.WriteFile(keyPath, serverKeyPEM, 0o600); err != nil {
		return fmt.Errorf("write nginx key %q: %w", keyPath, err)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
