---
title: 'Go'
draft: false
weight: 2
summary: Overview of Go client.
---

dyn.direct publishes a general purpose `DSDM` (Dynamic Sub Domain Management) client for Go.

#### Install

```bash
go get github.com/csnewman/dyndirect/go
```

#### Create Client

```go
c, err := dsdm.New(dsdm.DynDirect)
if err != nil {
    // ...
}
```

`dsdm.DynDirect` points to `v1.dyn.direct`.

#### Request Subdomain

```go
r, err := c.RequestSubdomain(ctx)
if err != nil {
    // ...
}

// r.Id, r.Domain, r.Token
```

The `RequestSubdomain` function will return a new dynamic subdomain.

- The format of the `Id` is an implementation detail and should not be parsed.
- The `Domain` will be of the format `<id>.<dsdm-server>`.
- The `Token` is a secret that can be used to manage the subdomain.

#### Dynamic Records

`IPv6` and `IPv4` records can be dynamically generated:

```go
dsdm.GetDomainForIP(r.Domain, net.ParseIP("127.0.0.1"))
```

Note: `GetDomainForIP` is a client side helper, and does not trigger a API request.

#### Set ACME Challenge

Wildcard SSL certificates can be acquired via the `DNS-01` challenge format. `dyn.direct` is not a certificate
authority and instead exposes an API to specify the `_acme-challenge.<id>.<dsdm-server>` record. This allows you to
acquire a certificate via any ACME compatible certificate authority with wildcard and `DNS-01` support, such as
[Let's Encrypt](https://letsencrypt.org/).

You can verify that `dyn.direct` has not covertly issued a certificate for your subdomain by checking a Certificate
Transparency Log, such as via [crt.sh](https://crt.sh/).

```go
err := c.SetSubdomainACMEChallenge(ctx, dsdm.SubdomainACMEChallengeRequest{
    ID:    r.Id,
    Token: r.Token,
    Values: []string{
        "my-challenge-token",
    },
})
if err != nil {
    // ...
}
```

The challenge token will expire after some period of time. You should not rely on this value being available for any
extended period.
