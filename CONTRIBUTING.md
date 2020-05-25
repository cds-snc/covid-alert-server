# Contributing

Thank you for considering contributing to COVID Shield!

We’d love to get your issues (if you find any bugs) and PRs (if you have any fixes)!

First, please review this document and the [Code of Conduct](CODE_OF_CONDUCT.md).

# Reporting security issues

COVID Shield takes security very seriously. In the interest of coordinated disclosure,
we request that any potential vulnerabilities be reported privately in accordance with
our [security policy](SECURITY.md).

## Contributing documentation and non-code changes

If you'd like to contribute a documentation or static file change, please
feel free to fork the project in GitHub and open a PR from that fork against
this repository.

## Contributing code

If you'd like to contribute code changes, the following steps will help you
setup a local development environment. If you're a Shopify employee, `dev up`
will install the above dependencies and `dev {build,test,run,etc.}` will work
as you'd expect.

Once you're happy with your changes, please fork the repository and push your
code to your fork, then open a PR against this repository.

## Development Environment via docker-compose

1. Fork https://github.com/CovidShield/server to your account
1. Clone your fork of CovidShield/server repo locally by running `git clone https://github.com/<username>/server.git`
1. Enter the repo directory `cd server`
1. Run `docker-compose up`

Note: It is normal to see a few errors from the retrieval service exiting initially while the MySQL database is instantiated

## Developing Locally

### Prerequisites

* Go (tested with 1.14)
* Ruby (tested with 2.6.5)
* Bundler
* [protobuf](https://developers.google.com/protocol-buffers/) (tested with libprotoc 3.11.4)
* [protoc-gen-go](https://github.com/golang/protobuf) (may only be needed to change `proto/*`)
* libsodium
* docker-compose
* MySQL

### Building

Run `make` or `make release` to build a release version of the servers.

### Running

```bash
# example...
export DATABASE_URL="root@tcp(localhost)/covidshield"
export KEY_CLAIM_TOKEN=thisisatoken=ON

./key-retrieval migrate-db

PORT=8000 ./key-submission
PORT=8001 ./key-retrieval
```

### Running Tests

If you're at Shopify, `dev up` will configure the database for you. If not
you will need to point to your database server using the environment variables
(note that the database will be clobbered so ensure that you don't point to a
production database):

```shell
$ export DB_USER=<username>
$ export DB_PASS=<password>
$ export DB_HOST=<hostname>
$ export DB_NAME=<test database name>
```

Then, ensure the appropriate requirements are installed:

```shell
$ bundle install
```

Finally, run:
```shell
$ make test
```
