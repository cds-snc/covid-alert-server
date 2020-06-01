# COVID Shield Diagnosis Server

![Container Build](https://github.com/CovidShield/server/workflows/Container%20Builds/badge.svg)

This repository implements a *Diagnosis Server* to use as a server for Apple/Google's [Exposure
Notification](https://www.apple.com/covid19/contacttracing) framework, informed by the [guidance
provided by Canada's Privacy
Commissioners](https://priv.gc.ca/en/opc-news/speeches/2020/s-d_20200507/).

The choices made in implementation are meant to maximize privacy, security, and performance. No
personally-identifiable information is ever stored, and nothing other than IP address is available to the server. No data at all is retained past 21 days. This server is designed to handle
use by up to 38 million Canadians, though it can be scaled to any population size.

In this document:

- [Overview](#overview)
   - [Retrieving _Diagnosis Keys_](#retrieving-diagnosis-keys)
   - [Retrieving _Exposure Configuration_](#retrieving-exposure-configuration)
   - [Submitting _Diagnosis Keys_](#submitting-diagnosis-keys)
- [Data usage](#data-usage)  
- [Generating one-time codes](#generating-one-time-codes)  
- [Protocol documentation](#protocol-documentation)
- [Deployment notes](#deployment-notes)
- [Contributing](#contributing)   
    1. [Set up a local development environment](#env-setup)   
    2. [Develop locally](#dev-local)  
    3. [Run tests](#run-tests) 
- [Who Built COVID Shield?](#who-built-covid-shield)

## Overview

_[Apple/Google's Exposure Notification](https://www.apple.com/covid19/contacttracing) specifications
provide important information to contextualize the rest of this document._

There are two fundamental operations conceptually:

* **Retrieving _Diagnosis Keys_**: retrieving a list of all keys uploaded by other users; and
* **Submitting _Diagnosis Keys_**: sharing keys returned from the EN framework with the server.

These two operations are implemented as two separate servers (`key-submission` and `key-retrieval`)
generated from this codebase, and can be deployed independently as long as they share a database. It
is also possible to deploy any number of configurations for each of these components, connected to
the same database, though there would be little value in deploying multiple configurations of
`key-retrieval`.

### Retrieving _Diagnosis Keys_

When _Diagnosis Keys_ are uploaded, the `key-submission` server stores the data defined and required
by the Exposure Notification API in addition to the time at which the data was received by the
server. This submission timestamp is rounded to the nearest hour for privacy preservation (to
prevent correlation of multiple keys to the same user).

The hour of submission is used to group keys into buckets, in order to prevent clients (the
soon-to-be-released _COVID Shield_ mobile app) from having to download a given set of key data
multiple times in order to repeatedly check for exposure.

The published _Diagnosis Keys_ are fetched—with some best-effort authentication—from a Content
Distribution Network (CDN), backed by `key-retrieval`. This allows a functionally-arbitrary number
of concurrent users.

### Retrieving _Exposure Configuration_

[_Exposure Configuration_](https://developer.apple.com/documentation/exposurenotification/enexposureconfiguration),
used to determine the risk of a given exposure, is also retrieved from the `key-retrieval` server. A JSON
document describing the current exposure configuration for a given region is available at the path
`/exposure-configuration/<region>.json`, e.g. for Ontario (region `ON`):

```sh
$ curl https://retrieval.covidshield.app/exposure-configuration/ON.json
{"minimumRiskScore":0,"attenuationLevelValues":[1,2,3,4,5,6,7,8],"attenuationWeight":50,"daysSinceLastExposureLevelValues":[1,2,3,4,5,6,7,8],"daysSinceLastExposureWeight":50,"durationLevelValues":[1,2,3,4,5,6,7,8],"durationWeight":50,"transmissionRiskLevelValues":[1,2,3,4,5,6,7,8],"transmissionRiskWeight":50}
```

### Submitting _Diagnosis Keys_

In brief, upon receiving a positive diagnosis, a health care professional will generate a _One Time
Code_ through a web application frontend (a reference implementation will be open-sourced soon), which
communicates with `key-submission`. This code is sent to the patient, who enters the code into their
(soon-to-be-released) _COVID Shield_ App. This code is used to authenticate the
Application (once) to the _Diagnosis Server_. Encryption keypairs are exchanged by the Application
and the `key-submission` server to be stored for fourteen days, and the One Time Code is immediately
purged from the database.

These keypairs are used to encrypt and authorize _Diagnosis Key_ uploads for the next fourteen
days, after which they are purged from the database.

The encryption scheme employed for key upload is _NaCl Box_ (a public-key encryption scheme using
Curve25519, XSalsa20, and Poly1305). This is widely regarded as an exceedingly secure implementation
of Elliptic-Curve cryptography.

## Data usage

The _Diagnosis Key_ retrieval protocol used in _COVID Shield_ was designed to restrict the data
transfer to a minimum. With large numbers of keys and assuming the client fetches using compression,
there is minimal protocol overhead on top of the key data size of 16 bytes.

In all examples below:

* Each case may generate up to 28 keys.
* Keys are valid and distributed for 14 days.
* Each key entails just under 18 bytes of data transfer when using compression.
* Key metadata and protocol overhead should in reality be minimal, but:
* Assume 50% higher numbers than you see below to be on the safe side. This README will be updated
  soon with more accurate real-world data sizes.

**Data below is current at May 12, 2020**. For each case, we assume the example daily new cases is a
steady daily recurrence.

### Deployed only to province of Ontario

There were 350 new cases in Ontario on May 10, 2020. 350 * 28 * 18 = 170kB per day, thus, deploying
to the province of Ontario at current infection rates would cause **7.1kB of download each hour**.

### Deployed to Canada

There were 1100 new cases in Canada on May 10, 2020. 1100 * 28 * 18 = 540kB per day, thus,
deploying to Canada at current infection rates would cause **23kB of download each hour**.

### Deployed to entire United States of America

There were 18,000 new cases in America on May 10, 2020. 18,000 * 28 * 18 = 8.9MB per day, thus,
deploying to the all of America at current infection rates would cause: **370kB of download each
hour**.

### Deployed to entire world

If _COVID Shield_ were deployed for the entire world, we would be inclined to use the "regions"
built into the protocol to implement key namespacing, in order to not serve up the entire set of
global _Diagnosis Keys_ to each and every person in the world, but let's work through the number in
the case that we wouldn't:

There were 74,000 new cases globally on May 10, 2020. 74,000 * 28 * 16 = 36MB per day, thus,
deploying to the entire world at current infection rates would cause: **1.5MB of download each
hour**.

## Generating one-time codes

We use a one-time code generation scheme that allows authenticated case workers to issue codes,
which are to be passed to patients with positive diagnoses via whatever communication channel is
convenient.

This depends on a separate service, holding credentials to talk to this (`key-submission`) server.
We have a sample implementation we will open source soon, but we anticipate that health authorities
will prefer to integrate this feature into their existing systems. The integration is extremely
straightforward, and we have [minimal examples in several
languages](https://github.com/CovidShield/server/tree/master/examples/new-key-claim). Most
minimally:

```bash
curl -XPOST -H "Authorization: Bearer $token" "https://submission.covidshield.app/new-key-claim"
```

## Protocol documentation

For a more in-depth description of the protocol, please see [the "proto" subdirectory of this
repo](/proto).

## Deployment notes

- `key-submission` depends on being deployed behind a firewall (e.g. [AWS
WAF](https://aws.amazon.com/waf/)), aggressively throttling users with 400 and 401 responses.

- `key-retrieval` assumes it will be deployed behind a caching reverse proxy.

We hope to provide reference implementations on AWS, GCP, and Azure via [Hashicorp Terraform](https://www.terraform.io/).

See [AWS Reference Implementation](config/infrastructure/aws/README.md) for more information.

## Contributing

Before you begin to contribute, see [_CONTRIBUTING.md_](CONTRIBUTING.md).

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

./key-retrieval migrate-db

PORT=8000 ./key-submission
PORT=8001 ./key-retrieval
```

Note that 302 is a [MCC](https://www.mcc-mnc.com/): 302 represents Canada.

<h3 id="run-tests">3. Run tests</h3>

If you're not a Shopify employee, you'll need to point to your database server using the environment variables
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

If you're a Shopify employee, `dev up` will configure the database for you and install the above dependencies and `dev {build,test,run,etc.}` will work as you'd expect.

Once you're happy with your changes, please fork the repository and push your code to your fork, then open a pull request against this repository.

## Who Built COVID Shield?

We are a group of Shopify volunteers who want to help to slow the spread of COVID-19 by offering our
skills and experience developing scalable, easy to use applications. We are releasing COVID Shield
free of charge and with a flexible open-source license.

For questions, we can be reached at <press@covidshield.app>.
