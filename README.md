# JWT Tool 🚀

This is a command-line tool for decoding JSON Web Tokens (JWTs). It allows you to decode and inspect JWT tokens either provided directly as strings or stored in files.

## Features 💡

- Decode JWT tokens and print their contents in JSON format.
- Supports decoding JWT tokens provided as strings or stored in files.

## Installation 🛠️ 

To install the JWT tool, you can simply use the following command.

```bash
go install "github.com/iaakanshff/jwt@latest"
```
## Usage 📝

```bash
Usage: jwt [options]

Default flags:
  -t string
        JWT token string

Optional flags:
  -f FILE
        File containing JWT tokens
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
$ jwt -f tokens.txt

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
    "typ": "JWT"
  },
  "payload": {
    "data_key": "password",
    "iat": 1516239022,
    "name": "Carmin",
    "payload": "the text of the token in token String"
  },
  "signature": "hHTLdOb-fBXZPzO4E1brpwzB2Cx28hBBRk54NWXTAAU"
}

```

## But Why Use Our Tool❓ 

We understand and appreciate that jwt.io provides a web-based solution for decoding JWT tokens, however our CLI tool offers the advantage of offline decoding. By keeping your tokens local, there's no risk of sensitive information landing in a website's database, ensuring your data remains secure and private. Yes, we are talking about you privacy-conscious users.

## Contributing 🤝

Contributions are welcome! If you have any suggestions, bug reports, or feature requests, feel free to open an issue or submit a pull request.
