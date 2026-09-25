# 0008 — Members serve TLS from first start; bootstrap pins a fingerprint

**Status:** accepted

A member started with `--tls` serves a throwaway self-signed certificate
until it has credentials, and writes its fingerprint next to the one-time
bootstrap code. `dh cp bootstrap` / `dh cp add-member` pin that fingerprint.
Once credentials are stored, the member serves its root-issued certificate
without a restart. `dh init` enables TLS by default.

**Why.** The previous flow bootstrapped over plain HTTP. Anyone on the path
could observe the one-time code and the credentials, and later every bearer
capability. Pinning needs no prior PKI and no second channel beyond the one
that already carries the code.

**Consequences.** Hosts receive the root CA in their join token and use HTTPS
from their first request. Federation agreements carry the grantor's root CA.
Plain HTTP remains for loopback development clusters. `tests/integration/tls_test.go`
covers the whole path.
