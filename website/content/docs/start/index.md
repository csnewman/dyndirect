---
title: 'Getting Started'
draft: false
weight: 1
summary: Overview of basic functionality via cURL.
---

dyn.direct uses a simple HTTP api called `DSDM` (Dynamic Sub Domain Management) to allow for automated subdomain
allocation and management.

#### Request Subdomain

```bash
curl --request POST --url https://v1.dyn.direct/subdomain
```

```json
{
  "id": "f7ba6402-2a47-4ba1-9e74-03f049cca41c",
  "domain": "f7ba6402-2a47-4ba1-9e74-03f049cca41c.v1.dyn.direct",
  "token": "<token-removed>"
}
```

The `subdomain` endpoint will return a new dynamic subdomain.

- The format of the `id` is an implementation detail and should not be parsed.
- The `domain` will be of the format `<id>.<dsdm-server>`.
- The `token` is a secret that can be used to manage the subdomain.

#### Dynamic Records

`IPv6` and `IPv4` records can be dynamically generated:

```bash
dig +short 127-0-0-1-v4.f7ba6402-2a47-4ba1-9e74-03f049cca41c.v1.dyn.direct A
127.0.0.1
```

```bash
dig +short 1-2-3-4-5-6-7-8-v6.f7ba6402-2a47-4ba1-9e74-03f049cca41c.v1.dyn.direct AAAA
1:2:3:4:5:6:7:8
```

#### Set ACME Challenge

Wildcard SSL certificates can be acquired via the `DNS-01` challenge format. `dyn.direct` is not a certificate
authority and instead exposes an API to specify the `_acme-challenge.<id>.<dsdm-server>` record. This allows you to
acquire a certificate via any ACME compatible certificate authority with wildcard and `DNS-01` support, such as
[Let's Encrypt](https://letsencrypt.org/).

You can verify that `dyn.direct` has not covertly issued a certificate for your subdomain by checking a Certificate
Transparency Log, such as via [crt.sh](https://crt.sh/).

```bash
curl --request POST \
  --url https://v1.dyn.direct/subdomain/f7ba6402-2a47-4ba1-9e74-03f049cca41c/acme-challenge \
  --header 'Content-Type: application/json' \
  --data '{
	"token": "<token-removed>",
	"values": [
		"your-challenge-token"
	]
}'
```

```bash
dig +short _acme-challenge.f7ba6402-2a47-4ba1-9e74-03f049cca41c.v1.dyn.direct TXT
"your-challenge-token"
```

The challenge token will expire after some period of time. You should not rely on this value being available for any
extended period.
