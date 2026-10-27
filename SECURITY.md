# Security Policy — rivet

## Supported versions

Rivet is pre-1.0 and has not cut a tagged release yet. Security fixes land on
`main`; test against the latest commit and report the SHA you used.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.** Instead:

1. Open a private [GitHub Security Advisory](https://github.com/GrayCodeAI/rivet/security/advisories/new), **or**
2. Email `security@graycodeai.com` with the details below.

Include in your report:

- A description of the vulnerability and the affected component.
- Steps to reproduce, ideally with a minimal proof-of-concept.
- The commit SHA you tested against.
- The potential impact and any suggested mitigation.

**Response targets:**

- Initial acknowledgement: within **48 hours**.
- Triage and severity assessment: within **5 business days**.
- Coordinated fix and disclosure: within **30 days** for high/critical, **90
  days** for medium/low (per industry-standard responsible disclosure).

## Disclosure policy

We follow [coordinated vulnerability disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure):

- Reporters receive credit in the advisory and `CHANGELOG.md` (unless they opt
  out).
- We ask that reporters refrain from public disclosure until a fix has been
  released or the disclosure deadline above has elapsed.
- We will not pursue legal action against good-faith researchers acting
  within this policy.

## Security practices in this repo

- **Standard library only:** there are no third-party Go dependencies, so the
  dependency surface is limited to the Go toolchain itself.
- **CI checks:** `gofmt`, `go vet`, and `go test -race` run on every push and
  pull request.
- **No secrets required:** the project needs no API keys or credentials, and
  none are stored in the repository.
- **Explicit execution only:** Rivet runs commands and blocks only when you
  invoke it. It does not install hooks or rewrite shell commands.
- **Hash-bound receipts:** a block's verification receipt is tied to the
  manifest, script, and check hashes; changing any of them invalidates the
  receipt.
- **Local code, not a sandbox:** blocks are shell scripts from the project.
  Review them before importing or running them. Verification is a quality
  gate, not an isolation boundary.
- **Private logs:** logs are written under a `0700` directory. Rivet makes no
  network requests.

## Scope

This policy covers the code in this repository and the build artefacts
published from it. It does not cover third-party tools or services, or misuse
by someone who already has shell access to the machine.
