# Security Policy

## Supported versions

This is a single-maintainer fork. Its version tracks the ACP schema release it
was generated from (see [Versioning](README.md#versioning)), not an independent
Go API line, and there is no capacity to backport across schema versions. Only
the latest published tag receives fixes. If you are on an older tag, upgrade
before reporting where that is possible at all.

## Reporting a vulnerability

Report privately, by e-mail, to **git@jinnoflife.com**. Please do not open a
public issue or pull request for a suspected vulnerability.

Include the schema or tag version in use, a minimal reproducing JSON-RPC
payload or Go snippet, and the impact you observed — panic, hang, memory
growth, or a message routed to the wrong handler.

If the issue reproduces against unmodified upstream code, please also report it
to [coder/acp-go-sdk](https://github.com/coder/acp-go-sdk); a fix there reaches
everyone, and this fork's policy is to push fixes upstream where they apply.

## What to expect

Maintained by one person alongside other work, so these are best-effort targets
rather than guarantees:

- Acknowledgement within 5 working days.
- Triage verdict — confirmed, not applicable, or needs more information —
  within 10 working days.
- A fix or a documented mitigation on a timeline that follows severity. You
  will be told the plan once it is triaged rather than left without an answer.

## Threat model

This library decodes JSON-RPC messages sent by a peer process — an agent or a
client — that is not necessarily trustworthy. Parsing whatever the peer sends,
including malformed and adversarial payloads, is the SDK's job, so the
following are in scope:

- Malformed, deeply nested or oversized JSON causing a panic, unbounded
  recursion, or excessive allocation during decode.
- Union and discriminator handling routing a payload to the wrong variant.
  Several fixes in this fork closed exactly that class of defect; see
  `CHANGELOG.md`.
- Resource exhaustion from a peer that never completes a message or opens
  unbounded concurrent requests.

Out of scope: transport security. ACP normally runs over a locally spawned
stdio process, so authentication and encryption of the channel belong to the
caller. Anyone exposing an ACP connection over a network is responsible for
securing that transport.
