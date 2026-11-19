package dna

// Version is the version of this package.
const Version = "2.0.0"

// UserAgent is sent with every request (see WithUserAgent). The API sits
// behind Cloudflare, which rejects requests without a descriptive agent.
const UserAgent = "DomainNameAPI-Go/" + Version + " (+https://github.com/domainreseller/go-dna)"
