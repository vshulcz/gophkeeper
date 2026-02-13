# Security

## Encryption Model
1. Secrets are encrypted on the client before transmission.
2. Encryption keys are derived from the user's password and a per-user random salt.
3. The server never receives plaintext data.

## Authentication
1. Users authenticate with username and password.
2. The server issues a signed token with a configurable TTL.
3. Clients clear local sessions on token expiration.

## Local Client Storage
Client stores are saved as JSON at:
`~/.config/gophkeeper/store.json` (platform-specific config directory).

The store contains:
1. Encrypted payloads
2. Sync metadata
3. Session token (cleared on logout or expiration)

## Threat Model Notes
1. This project is a reference implementation and does not claim to be a hardened vault.
2. Users should set a strong password and keep their device secure.
3. Token secrets must be set in production and not left at defaults.

## Reporting
If you discover a security issue, open a private report or contact the maintainer before disclosure.
