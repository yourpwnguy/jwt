<div align="center">

![Jwt LOGO](https://i.imgur.com/YcATkTY.png)

</div>
<h4 align="center">Simple, fast, and efficient tool for decoding, auditing, and verifying JWT tokens, fully offline.</h4>
<p align="center">
<img src="https://img.shields.io/github/go-mod/go-version/yourpwnguy/jwt">
<a href="https://github.com/yourpwnguy/jwt/graphs/contributors"><img src="https://img.shields.io/github/contributors-anon/yourpwnguy/jwt">
<a href="https://github.com/yourpwnguy/jwt/issues"><img src="https://img.shields.io/github/issues-raw/yourpwnguy/jwt">
<a href="https://github.com/yourpwnguy/jwt/stars"><img src="https://img.shields.io/github/stars/yourpwnguy/jwt">
</p>

---

`jwt` is a small terminal tool for working with JWT tokens. It decodes tokens, checks them for common security issues, and verifies signatures. Everything runs locally, so tokens never leave your machine.

## Features 💡

- Decode tokens from args, files, pipes, and logs.
- Tolerant input. `Bearer` prefixes, quotes, and extra whitespace are stripped automatically.
- Output modes for humans and scripts: `pretty`, `json`, `compact`, `table`.
- Single claim lookup with `--claim`.
- Passive security audit with severity levels and CI gates.
- Signature verification for HMAC, RSA, ECDSA, and EdDSA, with JWKS support.
- Stable exit codes for scripts and CI.

## Installation 🛠️

To install the jwt tool, you can simply use the following command.

```bash
go install -v "github.com/yourpwnguy/jwt/v2/cmd/jwt@latest"
cp ~/go/bin/jwt /usr/local/bin/
```

## Usage 📘

```bash
jwt [command] [options] [token]

Commands:
  decode   Decode tokens (default when no command is given)
  audit    Passive security checks, no network, no secrets
  verify   Check signature and exp, nbf, aud, iss
  version  Print version and exit

Decode examples:
  jwt -t <token>                  decode one token
  jwt <token>                     same, positional
  echo $TOKEN | jwt               pipe
  curl -s api | jq -r .token | jwt
  jwt -tL tokens.txt              batch file, one token per line
  cat app.log | jwt --extract     find tokens hidden in logs
  jwt $TOKEN -o json | jq .payload
  jwt $TOKEN --claim sub          print one claim, useful for scripts
  jwt -tL list -o table           batch overview

Audit examples:
  jwt audit <token>               human readable triage
  jwt audit -tL list -o json | jq
  jwt audit <token> --fail-on high   exit 2 when high or critical findings exist

Verify examples:
  jwt verify <token> --secret s3cr3t
  jwt verify <token> --pubkey @key.pem --aud api --iss auth.example.com
  jwt verify <token> --jwks https://auth.example.com/.well-known/jwks.json

Options (decode):
  -t <token>      single token
  -tL <file>      file with one token per line
  --extract       scan text for embedded tokens
  -o, --format    pretty, json, compact, table (default pretty)
  --H             header only
  --P             payload only
  --claim a.b.c   single claim by dot path
  -q, --quiet     just data, no Token headers or status line
  --no-color      plain output
  --verify-exp    exit 2 when expired or not yet valid
  --leeway 30s    clock skew tolerance
```

Notes:
  Flags can go before or after the token. Errors go to stderr, so stdout
  stays clean for piping into `jq`. Exit codes are `0` for success, `1`
  for system errors such as bad flags or unreadable files, and `2` for
  validation failures such as an expired token or a breached audit gate.

### DECODE:

```bash
$ jwt -t "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

● HS256 · sub 1234567890 · ! no exp claim (never expires)
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "iat": 1516239022,
    "name": "John Doe",
    "sub": "1234567890"
  },
  "signature": "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}
```

The first line is a summary with the algorithm, subject, and expiry status. The full JSON follows below it.

```bash
$ jwt -t "$TOKEN" --claim sub
1234567890
```

---

### AUDIT:

```bash
$ jwt audit -t "$TOKEN"

✗ FAIL 1/8 · fail-on high
────────────────────────────────────────────────────
✗ HIGH  missing-exp      no exp claim, stolen tokens never expire
────────────────────────────────────────────────────
  ✓ none-alg · weak-alg · long-lived · jku-x5u · kid · sensitive-claim · time-skew (7 passed)
```

8 checks ran, 1 failed. This token has no expiry, so a stolen token stays valid indefinitely.

As a CI gate, it fails the build on high or critical findings:

```bash
$ jwt audit -t "$TOKEN" --fail-on high
$ echo $?
2
```

---

### VERIFY:

```bash
$ jwt verify -t "$TOKEN" --secret "your-256-bit-secret"

✓ Signature: VALID   · HS256
✓ Claims: VALID
────────────────────────────────────────
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "iat": 1516239022,
    "name": "John Doe",
    "sub": "1234567890"
  },
  "signature": "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}
```

---

# The long version, and why it is built this way

This part covers what happens under the hood. The quick start above is enough to use the tool. Everything below is the reasoning behind it.

The goal is a tool for terminals, where tokens actually live. API responses, log files, CI pipelines. That shaped most of the decisions here.

## Decoding is not verifying

Seeing what is inside a token and trusting a token are different things, so the code keeps them apart. `internal/jwt` only decodes, with no crypto and no network. `internal/verify` handles trust decisions and is the only place that touches secrets. The audit package only reads structure. It takes no secrets and makes no network calls, so it is safe to run on dumps that are not trusted yet.

## Forgiving input, strict parsing

Tokens arrive messy in practice. `Bearer` prefixes from headers, quotes from JSON, stray whitespace. The input layer strips all of that without complaining. Once the string is clean, parsing is strict. Three parts, base64url, JSON objects. Errors say which part broke.

## Human and machine output

Every command has a human mode and a machine mode. Pretty output has status lines, symbols, and colors. JSON and compact output have none of that, so `jwt -o json | jq` always works. Errors go to stderr for the same reason.

## Exit codes

`0` means success. `1` means a system error, such as bad flags or an unreadable file. `2` means validation failed, such as an expired token or a breached audit gate.

## Project layout

The code is split into small packages, each with a clear boundary:

- `cmd/jwt` is the thin entry point. It wires stdio and signals, then hands off.
- `internal/cli` owns flags, subcommands, and exit codes. No JWT logic lives here.
- `internal/jwt` owns decoding and claim helpers. No network, no crypto.
- `internal/input` owns where tokens come from. Args, files, stdin, Bearer stripping, extraction.
- `internal/output` owns rendering. The only place that prints to stdout.
- `internal/audit` owns the passive security rules. Pure functions, safe for concurrent use.
- `internal/verify` owns signatures and claims. The only place that touches secrets or network.
- `internal/reader` and `internal/version` are tiny helpers for files and version info.

## But why use our tool

jwt.io works well for a quick look at a single token. But production tokens should not be pasted into websites, and a browser cannot grep logs or fail builds. This tool runs locally, handles thousands of tokens at once, and fits into existing pipes.

Tools like `jwt-cli` and `jwt-hack` cover similar ground. This one prioritizes clean boundaries and reliability over feature count. Decode, audit, and verify answer different questions, so they stay separate.

## Contributing 🤝

Contributions are welcome. If you have suggestions, bug reports, or feature requests, feel free to open an issue or submit a pull request. The codebase is small on purpose, so it is easy to follow where a change should go.
