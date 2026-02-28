# Roadmap

## Planned

### Keychain support for secret storage (#2)
- OS keychain integration (macOS Keychain, Linux libsecret, Windows Credential Manager)
- Replace plaintext token storage in `.env.openclaw`
- Fallback to plaintext when no keychain available (CI, Docker-only hosts)
- Go libraries: `zalando/go-keyring` or `99designs/keyring`


### Let's Encrypt TLS cert support
- Flag like `--tls-domain example.com` to obtain a real TLS cert via certbot/acme.sh
- Replaces self-signed `nginx-cert.pem`/`nginx-key.pem` with a trusted cert
- Fail fast if certbot not installed on host
- Open questions:
  - DNS challenge vs HTTP challenge (HTTP needs port 80 open + DNS already pointed)
  - Renewal strategy: cron/timer on host vs compose sidecar container
  - Runs at `setup.sh` runtime (not generation time) since it needs the actual host
  - DNS challenge needs provider API credentials
