<div align="center">

![Jwt LOGO](https://i.imgur.com/YcATkTY.png)

</div>
<h4 align="center">Simple, fast, and efficient tool for decoding, editing, auditing, and verifying JWT tokens, fully offline.</h4>
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
- Live token editor with `jwt tui`: edit header and payload, watch the encoded token and audit update, sign HMAC tokens, copy the result, and move with Vim-style `ctrl+h/j/k/l` motions.
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
  tui      Live token editor, interactive
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

Editor examples:
  jwt tui <token>                 edit with live encoding and audit
  echo $TOKEN | jwt tui           edit a piped token
  jwt tui -f token.txt            edit one token from a file
  jwt tui                         start from a blank token

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

### TUI:

```bash
$ jwt tui "$TOKEN"
```

```text
  JWT.   T O K E N  P R E S S                                            ● HS256   ● UNTOUCHED   ● 1 ISSUE
  ────────────────────────────────────────────────────────────────────────────────────────────────────────

   █████    ██       1 {                                                 │  ENCODED TOKEN          155 B
   ██ ██   ███       2   "alg": "HS256",                                 │
   ██ ██    ██       3   "typ": "JWT"                                    │  eyJhbGciOiJIUzI1NiIsInR5cCI6
   ██ ██    ██       4 }                                                 │  IkpXVCJ9.eyJzdWIiOiIxMjM0NTY
   █████  █████                                                          │  3ODkwIiwibmFtZSI6IkpvaG4gRG9
                                                                         │  lIiwiaWF0IjoxNTE2MjM5MDIyfQ.
   HEADER                                                                │  SflKxwRJSMeKKF2QT4fwpMeJf36P
   ━━━━━━                                                                │  Ok6yJV_adQssw5c
                                                                         │
   OBJECT                                                                │  ────────────────────────────
   valid                                                                 │  ALG       HS256
   2 keys                                                                │  SIGNATURE untouched
   original                                                              │  EXPIRES   no exp claim
                                                                         │  SIZE      155 B
   ✓ 02 PAYLOAD                                                          │  ────────────────────────────
                                                                         │  AUDIT              ✗ 1 ISSUE
   ◇ 03 SIGNATURE                                                        │  ● ● ● ● ● ● ● ●
                                                                         │  ✗ HIGH  missing-exp
                                                                         │  no exp claim, stolen
                                                                         │  tokens never expire

  ────────────────────────────────────────────────────────────────────────────────────────────────────────
  ● EDIT   ctrl+hjkl move · tab section · ctrl+s sign · ctrl+y copy · ctrl+o command                L4  C2
```

The editor is a live scratchpad. Header and payload are plain JSON. The token panel shows the exact encoded token, the honest signature state, and the audit summary. Every valid keystroke rebuilds the token from the current segments and re-runs the offline audit.

The section rail, the page number, and the accent color follow the active section. Narrower terminals turn the rail into a tab row, and the narrowest move the token under the editor, so the same information survives at 60 columns.

| Key | Action |
| --- | --- |
| `arrows` | Move the cursor, always available |
| `ctrl+h` / `ctrl+j` / `ctrl+k` / `ctrl+l` | Move left / down / up / right, Vim style |
| `tab` / `shift+tab` | Next / previous section |
| `ctrl+o` or `esc` | Command mode |
| `ctrl+s` | Sign the token |
| `ctrl+y` | Copy the encoded token |
| `1` / `2` / `3` | Header / payload / signature (command mode) |
| `f` / `c` | Format / compact the current JSON (command mode) |
| `a` / `v` / `?` | Audit / full token / help (command mode) |
| `i` or `enter` | Back to editing (command mode) |
| `q` | Quit (command mode, or `ctrl+c` anywhere) |

The motions are on by default. `ctrl+i` is the Tab key at the terminal level, so it cannot be distinguished from Tab, and Tab keeps switching sections. `ctrl+h` and `ctrl+k` would otherwise be the textarea's delete aliases; Backspace and Delete still delete. If you would rather have the plain arrow-key editor, start with `jwt tui --no-vim-keys`.

`ctrl+o` is the recommended way into command mode because it is a single control key. `esc` works too, but a very fast next keystroke can look like an Alt sequence to the terminal.

The Signature tab is editable. You can paste a base64url signature produced by another tool. If the header or payload changes after a signature, the status turns `STALE` and copying warns you. `ctrl+s` re-signs HMAC tokens with the secret you enter. Asymmetric signing is not in this build, so RS, PS, ES, and EdDSA tokens can still be edited, audited, and copied, but their signature will be marked stale until you paste or generate a new one.

The editor never rewrites a token silently. Untouched panes keep their original base64 segments, and edited panes encode the exact text on screen, including whitespace and duplicate keys. That matters when you are testing how different parsers treat the same token. The editor shows pretty JSON, but the token panel is the real bytes.

Copy uses OSC 52, so it works over SSH in terminals that support it. Secrets and tokens stay in memory. Nothing is written to disk and nothing is sent over the network. Use `--theme auto|dark|light` if the terminal detects the background wrong, `--no-color` for a plain UI, and `--no-vim-keys` to turn off the `ctrl+h/j/k/l` motions.

The TUI is built with Bubble Tea, Bubbles, and Lip Gloss. They are pinned to versions that keep the project's Go 1.21 minimum. The core decode, audit, and verify paths do not change.

---

# The long version, and why it is built this way

This part covers what happens under the hood. The quick start above is enough to use the tool. Everything below is the reasoning behind it.

The goal is a tool for terminals, where tokens actually live. API responses, log files, CI pipelines. That shaped most of the decisions here.

## Decoding is not verifying

Seeing what is inside a token and trusting a token are different things, so the code keeps them apart. `internal/jwt` only decodes, with no crypto and no network. `internal/verify` handles trust decisions and is the only place that checks a signature or fetches keys. `internal/secret` owns secret loading, and `internal/sign` computes HMAC signatures for the editor. The audit package only reads structure. It takes no secrets and makes no network calls, so it is safe to run on dumps that are not trusted yet.

## Forgiving input, strict parsing

Tokens arrive messy in practice. `Bearer` prefixes from headers, quotes from JSON, stray whitespace. The input layer strips all of that without complaining. Once the string is clean, parsing is strict. Three parts, base64url, JSON objects. Errors say which part broke.

## Human and machine output

Every command has a human mode and a machine mode. Pretty output has status lines, symbols, and colors. JSON and compact output have none of that, so `jwt -o json | jq` always works. Errors go to stderr for the same reason.

## The editor keeps exact bytes

The TUI does not decode JSON, change the key order, and re-encode it. Untouched panes keep the original base64 segments from the token you opened. The pane you edit is encoded from the exact text on screen. If a token arrives with unusual whitespace, or even duplicate keys, the untouched panes still encode to the same bytes. That is useful when two services disagree about what a token says.

The visible JSON is pretty-printed for editing, but that view is not the source of truth for the token. The encoded pane is. The signature status is equally literal: `UNTOUCHED` means loaded and not modified, `SIGNED` means this session computed it, `STALE` means the header or payload changed after signing, and `UNSIGNED` means there is no signature or `alg` is `none`. The editor never calls a signature valid without a key.

## Exit codes

`0` means success. `1` means a system error, such as bad flags or an unreadable file. `2` means validation failed, such as an expired token or a breached audit gate. The TUI follows the same rule: quitting normally is `0`, a missing terminal or launch failure is `1`.

## Project layout

The code is split into small packages, each with a clear boundary:

- `cmd/jwt` is the thin entry point. It wires stdio and signals, then hands off.
- `internal/cli` owns flags, subcommands, and exit codes. No JWT logic lives here.
- `internal/jwt` owns decoding and claim helpers. No network, no crypto.
- `internal/input` owns where tokens come from. Args, files, stdin, Bearer stripping, extraction.
- `internal/output` owns rendering. The only place that prints to stdout.
- `internal/audit` owns the passive security rules. Pure functions, safe for concurrent use.
- `internal/verify` owns signatures and claims. The only place that checks trust or fetches keys.
- `internal/sign` computes HMAC signatures for the editor. No trust decisions.
- `internal/secret` loads HMAC secrets and rejects PEM key material, so verify and the editor share one confusion guard.
- `internal/tui` owns the interactive editor. Rendering and keyboard flow only.
- `internal/reader` and `internal/version` are tiny helpers for files and version info.

## But why use our tool

jwt.io works well for a quick look at a single token. But production tokens should not be pasted into websites, and a browser cannot grep logs or fail builds. This tool runs locally, handles thousands of tokens at once, and fits into existing pipes. The TUI keeps the same rule: you can edit and re-sign a token without it leaving your machine.

Tools like `jwt-cli` and `jwt-hack` cover similar ground. This one prioritizes clean boundaries and reliability over feature count. Decode, edit, audit, and verify answer different questions, so they stay separate.

## Contributing 🤝

Contributions are welcome. If you have suggestions, bug reports, or feature requests, feel free to open an issue or submit a pull request. The codebase is small on purpose, so it is easy to follow where a change should go.
