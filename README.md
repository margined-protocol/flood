# Flood

Liquidity Provider bot for CL pools.

![Flood][3]

Flood creates and manages Concentrated Liquidity (CL) pool positions ensuring that capital deployed is done according to a useful set of rules.

The primary objective is to try to maintain two positions of quote and base assets that allows the owner of the bot to define the price(s) at which the quote and base are to be sold.

## Strategy Description

The strategy looks at the current theoretical price of sqASSET and the market price of sqASSET and adjust two LP positions accordingly.

It follow a number of key steps:

1. Calculate market and theoretical price delta
2. See if delta is greater than a threshold
3. If threshold is passed then adjust the positions, if not do nothing

## Installation

Releases for Linux, Windows and Mac are available on the [releases page][4].

### Config

An example config file with comments is provided at
[`./configs/config.example.toml`][6]

### Usage

To run power balance pass the path of the configuration to the `-c` flag.

```sh
LOG_LEVEL=debug ./bin/flood -c configs/config.example.toml
```

### Managing keys

Power Balance can be configured to use [`pass`][5] as a keychain.

```toml
signer_account = "bot-1"

[key]
app_name = "osmosis"
backend = "pass"
root_dir = "/home/margined"
```

This will load a key from
`/home/margined/.password-store/keyring-osmosis/bot-1.info.gpg`, equivalent to
running

```sh
pass show keyring-osmosis/bot-1.info
```

<!-- dprint-ignore-start -->

> [!TIP]
> If you are using pass on a headless server set `default-cache-ttl 31446952`
> and `max-cache-ttl 31446952` in `~/.gnupg/gpg-agent.conf`.

<!-- dprint-ignore-end -->

[1]: https://github.com/margined-protocol/flood/actions/workflows/golangci-lint.yml/badge.svg
[2]: https://github.com/margined-protocol/flood/actions/workflows/golangci-lint.yml
[3]: assets/images/flood.webp
[4]: ../../releases
[5]: https://www.passwordstore.org/
[6]: configs/config.example.toml
