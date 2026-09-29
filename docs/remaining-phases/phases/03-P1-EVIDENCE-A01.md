# P1-EVIDENCE-A01 — Signed Evidence

EvidenceRecord binds qualification_id, campaign_id, source_sha, verifier_source_sha, node_id, event_type, UTC observation, monotonic sequence, payload digest, previous digest, signer public key and Ed25519 signature.

Signing uses a private key; public keys verify only. Never package private keys.

MANIFEST.sha256 covers evidence artifacts. Fresh verifier rejects missing/extra decisive artifacts, hash mismatch, malformed record, wrong signer, wrong qualification/source, replay and signature failure.

Negative control: mutate one byte in a copied decisive artifact; verifier MUST fail.
