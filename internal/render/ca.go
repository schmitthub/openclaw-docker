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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// generateTLSCert creates a self-signed TLS certificate for the Envoy
// ingress listener. Existing certs are preserved across re-runs.
func generateTLSCert(opts Options) error {
	certPath := filepath.Join(opts.OutputDir, "compose", "envoy", "server-cert.pem")
	keyPath := filepath.Join(opts.OutputDir, "compose", "envoy", "server-key.pem")

	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate TLS key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial number: %w", err)
	}

	dnsNames := []string{"localhost", "openclaw-gateway", "envoy"}
	ipAddresses := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.28.0.2"), net.ParseIP("::1")}

	// Include the external origin hostname in SANs for production deployments.
	if opts.ExternalOrigin != "" {
		origin := opts.ExternalOrigin
		if !strings.Contains(origin, "://") {
			origin = "https://" + origin
		}
		if u, err := url.Parse(origin); err == nil && u.Hostname() != "" {
			host := u.Hostname()
			if ip := net.ParseIP(host); ip != nil {
				ipAddresses = append(ipAddresses, ip)
			} else {
				dnsNames = append(dnsNames, host)
			}
		}
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "openclaw-gateway",
		},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(2 * 365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("create TLS certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal TLS key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if opts.ConfirmWrite != nil {
		if err := opts.ConfirmWrite(certPath); err != nil {
			return err
		}
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("write TLS cert %q: %w", certPath, err)
	}

	if opts.ConfirmWrite != nil {
		if err := opts.ConfirmWrite(keyPath); err != nil {
			return err
		}
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write TLS key %q: %w", keyPath, err)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
