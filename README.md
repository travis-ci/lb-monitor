# travis-ci/lb-monitor

A mini monitor to keep track of whether all load balancer IPs are available. It
performs a DNS resolution, attempts to connect to port `443` (HTTPS) of each IP
and reports if it cannot connect.

Also optionally reports counts to librato.

## Settings

* `HOSTNAME` - the hostname to query, e.g. `travis-ci.org`.
* `UPSTREAM_HOSTNAME` - a second hostname to check IPs against, e.g. `travis-ci.com.herokudns.com`.
* `POLL_INTERVAL` - the number of seconds to wait in between polls. Defaults to `60` seconds.
* `DIAL_TIMEOUT` - the number of seconds to wait for an answer until the TCP connection on port `443` times out. Defaults to `5` seconds.
* `LIBRATO_USER` - (optional) the librato user, usually looks like an email address.
* `LIBRATO_TOKEN` - (optional) the librato token.
* `LIBRATO_SOURCE` - (optional) the librato source. If none is provided, it will attempt to use the `DYNO` env var. If that is empty, it will use the hostname of the machine running the monitor.

## Install

    $ go get -u github.com/FiloSottile/gvt
    $ gvt restore

## Running

    $ go run main.go
