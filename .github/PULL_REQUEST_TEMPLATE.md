## Summary
<!-- What does this PR do and why? Keep it short. -->

Closes #<!-- sub-issue number -->

## Checklist
- [ ] Branch is named `<type>/<issue-code>-<slug>` (e.g. `feat/b3.1-evaluate`)
- [ ] CI is green (Go, lint, web, compliance, docker, security)
- [ ] Definition of Done of the linked sub-issue is ticked
- [ ] Tests added/updated for the behavior changed
- [ ] No secret keys, `.env`, or real credentials committed (only TEST ONLY fixtures)
- [ ] Server code does not reference `SecretKey` / `Decryptor`; the secret key never leaves the browser
- [ ] Sensitive values (balance, amount, quantity, total) appear only as ciphertext in requests, responses, DB, and logs
- [ ] CKKS parameters live in `config/`, not in code
- [ ] Docs updated if a contract changed (`docs/API_CONTRACT.md`, `docs/CRYPTO_BRIDGE.md`, `docs/PARAMS.md`)

## Notes for reviewers
<!-- Anything non-obvious: trade-offs, follow-ups, how to test manually. -->
