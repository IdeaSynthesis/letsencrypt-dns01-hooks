# letsencrypt-dns01-hooks
DNS provider hooks for the letsencrypt.sh script's dns01 ACME challenge

https://github.com/lukas2511/letsencrypt.sh provides a BASH script that only uses OpenSSL/cURL and similar tools to retrieve letsencrypt certificates. It also provides support for the dns01 ACME challenge (which quite handily lets a certificate be requested and retrieved without having to write files anywhere, only requiring a DNS TXT record). To handle different DNS provider APIs, letsencrypt.sh delegates the DNS operations to a hook script. This repository contains a hook script (written in Go) that implements the hook command line interface.

## Why

I use Linode's DNS servers for some of my work, and wanted to run a single bastion host that is responsible for all my cert requests and then just distributes them to the individual servers (that way I can cleanly handle retrieving a cert on a single machine and then deploying that single cert to a pool of servers sharing a domain name). There wasn't a likely candidate at https://github.com/lukas2511/letsencrypt.sh/wiki/Examples-for-DNS-01-hooks, so I made one.

## How It Works

hook.go currently implements support for the Linode DNS API: it will retrieve an API key from the environment variable LINODE_API_KEY, then based on the domain you're requesting will check the Linode domains list accessible by that API key for the matching one. Once found, it will check for a TXT record with the name _acme-challenge.&lt;domain name&gt;, and if one exists it will update it; otherwise it will create it. It will wait until the new text record properly resolves on the local interface (Linode can take up to 15 minutes to reload their nameservers and have it propagate) and then it cleans up the record once letsencrypt is done with it.

## Building

Developed and tested on Ubuntu Linux 14.04 with Go 1.6 (earlier versions may work: we're not doing anything special here). Make sure that Go is installed and the workspace is setup with GOPATH properly defined. Then:

- git clone https://github.com/IdeaSynthesis/letsencrypt-dns01-hooks $GOPATH/src/github.com/IdeaSynthesis/letsencrypt-dns01-hooks
- cd $GOPATH/src/github.com/IdeaSynthesis/letsencrypt-dns01-hooks
- go install

## Usage Requirements

- Requires https://github.com/lukas2511/letsencrypt.sh be installed, with all the dependencies.
- Requires a Linode API key.

## Usage

Request a certificate using dns01 challenge deployment: the required key is passed in via environment variable.

    LINODE_API_KEY=<API key from https://manager.linode.com/profile/api> <path to letsencrypt.sh> -c --out <path to output folder> --algo rsa --challenge dns-01 -d <DOMAIN> -k $GOPATH/bin/letsencrypt-dns01-hooks

The hook will create or update the entry, then block until the DNS entry propagates. Unfortunately letsencrypt.sh (and I'm assuming letsencrypt's server in general) doesn't support completely splitting the request and the challenge phase, so we have to twiddle our thumbs until we can be certain the DNS record has been created.

## Future Work

Update the implementation to choose an API based on the environment variables (should support any DNS provider with an API: at the least Route 53, which is the other DNS provider I use).

## Issues

Let us know at https://github.com/IdeaSynthesis/letsencrypt-dns01-hooks/issues of any issues you find.
