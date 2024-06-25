<div align="center">

![GoStyle LOGO](https://i.imgur.com/YcATkTY.png)

</div>
<h4 align="center">Simple, fast, and efficient tool for decoding JWT tokens, supporting base64 and JSON Web Token (JWT) formats..</h4>
<p align="center">
<img src="https://img.shields.io/github/go-mod/go-version/iaakanshff/crtfinder">
<!-- <a href="https://github.com/iaakanshff/crtfinder/releases"><img src="https://img.shields.io/github/downloads/iaakanshff/crtfinder/total"> -->
<a href="https://github.com/iaakanshff/crtfinder/graphs/contributors"><img src="https://img.shields.io/github/contributors-anon/iaakanshff/crtfinder">
<!-- <a href="https://github.com/iaakanshff/crtfinder/releases/"><img src="https://img.shields.io/github/release/iaakanshff/crtfinder"> -->
<a href="https://github.com/iaakanshff/crtfinder/issues"><img src="https://img.shields.io/github/issues-raw/iaakanshff/crtfinder">
<a href="https://github.com/iaakanshff/crtfinder/stars"><img src="https://img.shields.io/github/stars/iaakanshff/crtfinder">
<!-- <a href="https://github.com/iaakanshff/crtfinder/discussions"><img src="https://img.shields.io/github/discussions/iaakanshff/crtfinder"> -->
</p>

## Features 💡

- Decode JWT tokens and print their contents in JSON format.
- Supports decoding JWT tokens provided as strings or stored in files.

## Installation 🛠️ 

To install the JWT tool, you can simply use the following command.

```bash
go install -v "github.com/iaakanshff/jwt/cmd/jwt@latest"
cp ~/go/bin/jwt /usr/local/bin/
```
## Usage 📝

```yaml
Usage: jwt [options]

Options: [flag] [argument] [Description]

INPUT:
  -t string     JWT token string to decode
  -tL FILE      File containing list of JWT tokens

DEBUG:
  -v none       Check current version
```

## Examples 📄

- Decoding a JWT String

```bash
$ jwt -t "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

Token 1:
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

<br>

- Decode JWT tokens stored in a file:

```bash
$ jwt -tL tokens.txt

Token 1:
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "data_key": "secret",
    "iat": 1516239022,
    "name": "Breake",
    "payload": "the text of the token in Token String"
  },
  "signature": "d2wZRIuXyfzjs4tXz7DfQgnB80heGvZoBIt1x4lIvZM"
}

Token 2:
{
  "header": {
    "alg": "HS256",
    "kid": "12345",
    "typ": "JWT"
  },
  "payload": {
    "aud": [
      "example.com",
      "another-example.com"
    ],
    "custom_claim": "custom_value",
    "email": "john.doe@example.com",
    "exp": 1716239022,
    "iat": 1616239022,
    "iss": "example.com",
    "jti": "unique-token-id",
    "metadata": {
      "device": "iPhone",
      "ip": "192.168.1.1",
      "location": "San Francisco"
    },
    "name": "John Doe",
    "nbf": 1616239022,
    "permissions": {
      "admin": [
        "read",
        "write",
        "delete"
      ],
      "user": [
        "read"
      ]
    },
    "preferences": {
      "language": "en-US",
      "theme": "dark"
    },
    "roles": [
      "admin",
      "user"
    ],
    "sub": "user123"
  },
  "signature": "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}

```

## But Why Use Our Tool❓ 

We understand and appreciate that jwt.io provides a web-based solution for decoding JWT tokens, however our CLI tool offers the advantage of decoding offline. By keeping your tokens local, there's no risk of sensitive information landing in a website's database, ensuring your data remains secure and private. Yes, we are talking about you privacy-conscious users.

## Contributing 🤝

Contributions are welcome! If you have any suggestions, bug reports, or feature requests, feel free to open an issue or submit a pull request.
