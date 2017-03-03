package certtest

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/square/certstrap/pkix"
)

const (
	o        = "certtest Organization"
	ou       = "certtest Unit"
	country  = "AQ"
	province = "Ross Island"
	city     = "McMurdo Station"

	keySize = 2048
)

// Authority represents a Certificate Authority. It should not be used for
// anything except ephemeral test usage.
type Authority struct {
	cert *pkix.Certificate
	key  *pkix.Key
}

// BuildCA creates a new test Certificate Authority. The name argument can be
// used to distinguish between multiple authorities.
func BuildCA(name string) (*Authority, error) {
	key, err := pkix.CreateRSAKey(keySize)
	if err != nil {
		return nil, err
	}

	crt, err := pkix.CreateCertificateAuthority(key, ou, 1, o, country, province, city, name)
	if err != nil {
		return nil, err
	}

	return &Authority{
		cert: crt,
		key:  key,
	}, nil
}

// SignOption is used to alter the signed certificate parameters.
type SignOption func(*signOptions)

// WithIPs adds the passed IPs to be valid for the requested certificate.
func WithIPs(ips ...net.IP) SignOption {
	return func(options *signOptions) {
		options.ips = ips
	}
}

// WithDomains adds the passed domains to be valid for the requested
// certificate.
func WithDomains(domains ...string) SignOption {
	return func(options *signOptions) {
		options.domains = domains
	}
}

// BuildSignedCertificate creates a new signed certificate which is valid for
// `localhost` and `127.0.0.1` by default. This can be changed by passing in
// the various options. The certificates it creates should only be used
// ephemerally in tests.
func (a *Authority) BuildSignedCertificate(name string, options ...SignOption) (*Certificate, error) {
	key, err := pkix.CreateRSAKey(keySize)
	if err != nil {
		return nil, err
	}

	opts := defaultSignOptions()
	for _, o := range options {
		opts.apply(o)
	}

	csr, err := pkix.CreateCertificateSigningRequest(key, ou, opts.ips, opts.domains, o, country, province, city, name)
	if err != nil {
		return nil, err
	}

	crt, err := pkix.CreateCertificateHost(a.cert, a.key, csr, 1)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		cert: crt,
		key:  key,
	}, nil
}

// CertificatePEM returns the authorities certificate as a PEM encoded bytes.
func (a *Authority) CertificatePEM() ([]byte, error) {
	return a.cert.Export()
}

// CertPool returns a certificate pool which is pre-populated with the
// Certificate Authority.
func (a *Authority) CertPool() (*x509.CertPool, error) {
	cert, err := a.CertificatePEM()
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)

	return pool, nil
}

// Certificate represents a Certificate which has been signed by a Certificate
// Authority.
type Certificate struct {
	cert *pkix.Certificate
	key  *pkix.Key
}

// TLSCertificate returns the certificate as Go standard library
// tls.Certificate.
func (c *Certificate) TLSCertificate() (tls.Certificate, error) {
	certBytes, err := c.cert.Export()
	if err != nil {
		return tls.Certificate{}, nil
	}

	keyBytes, err := c.key.ExportPrivate()
	if err != nil {
		return tls.Certificate{}, nil
	}

	return tls.X509KeyPair(certBytes, keyBytes)
}

type signOptions struct {
	domains []string
	ips     []net.IP
}

func defaultSignOptions() *signOptions {
	return &signOptions{
		domains: []string{},
		ips:     []net.IP{},
	}
}

func (s *signOptions) apply(option SignOption) {
	option(s)
}
