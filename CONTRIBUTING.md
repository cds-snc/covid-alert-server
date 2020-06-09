# Contributing Guidelines

Welcome, and thank you for considering contributing to COVID Shield!

- [Code of Conduct](#code-of-conduct)
- [Reporting Security Issues](#reporting-security-issues)
- [Contributing Documentation](#contributing-documentation)
- [Contributing Code](#contributing-code)
    1. [Set up a local development environment](#env-setup)   
    2. [Develop locally](#dev-local)  
    3. [Run tests](#run-tests)

## Code of Conduct

First, please review this document and the [Code of Conduct](CODE_OF_CONDUCT.md).

## Reporting Security Issues

COVID Shield takes security very seriously. In the interest of coordinated disclosure,
we request that you report any potential vulnerabilities privately in accordance with
our [_Security Policy_](SECURITY.md).

## Contributing Documentation

If you'd like to contribute a documentation or static file change, feel free to fork the project in GitHub and open a pull request from that fork against this repository.

## Contributing Code

<h3 id="env-setup">1. Set up a local development environment</h3>

#### Development environment via docker-compose

1. Fork https://github.com/CovidShield/server to your account.
2. Clone your fork of the **CovidShield/server** repo locally by running `git clone https://github.com/<username>/server.git`.
3. Enter the repo directory `cd server`.
4. Run `docker-compose up`.

**Note**: It is normal to see a few errors from the retrieval service exiting initially while the MySQL database is instantiated

<h3 id="dev-local">2. Develop locally</h3>

#### Prerequisites

* Go (tested with 1.14)
* Ruby (tested with 2.6.5)
* Bundler
* [protobuf](https://developers.google.com/protocol-buffers/) (tested with libprotoc 3.11.4)
* [protoc-gen-go](https://github.com/golang/protobuf) (may only be needed to change `proto/*`)
* libsodium
* docker-compose
* MySQL

#### Building

Run `make` or `make release` to build a release version of the servers.

#### Running

```bash
# example...
export DATABASE_URL="root@tcp(localhost)/covidshield"
export KEY_CLAIM_TOKEN=thisisatoken=302

./build/release/key-retrieval migrate-db

PORT=8000 ./build/release/key-submission
PORT=8001 ./build/release/key-retrieval
```

Note that 302 is a [MCC](https://www.mcc-mnc.com/): 302 represents Canada.

<h3 id="run-tests">3. Run tests</h3>

Set your database connection details using environment variables
(note that the database will be clobbered so ensure that you don't use a
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

Once you're happy with your changes, please fork the repository and push your code to your fork, then open a pull request against this repository.
