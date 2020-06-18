# COVID Shield Diagnosis Server

![Container Build](https://github.com/CovidShield/server/workflows/Container%20Builds/badge.svg)

Adapted from <https://github.com/CovidShield/server> ([see changes](https://github.com/cds-snc/covid-shield-server/blob/master/FORK.md))

This repository implements a diagnosis server to use as a server for Apple/Google's [Exposure
Notification](https://www.apple.com/covid19/contacttracing) framework, informed by the [guidance
provided by Canada's Privacy
Commissioners](https://priv.gc.ca/en/opc-news/speeches/2020/s-d_20200507/).

The choices made in implementation are meant to maximize privacy, security, and performance. No
personally-identifiable information is ever stored, and nothing other than IP address is available to the server. No data at all is retained past 21 days. This server is designed to handle
use by up to 38 million Canadians, though it can be scaled to any population size.

In this document:

- [Overview](#overview)
   - [Retrieving diagnosis keys](#retrieving-diagnosis-keys)
   - [Retrieving Exposure Configuration](#retrieving-exposure-configuration)
   - [Submitting diagnosis keys](#submitting-diagnosis-keys)
- [Data usage](#data-usage)
- [Generating one-time codes](#generating-one-time-codes)
- [Protocol documentation](#protocol-documentation)
- [Deployment notes](#deployment-notes)
- [Metrics and Tracing](#metrics-and-tracing)
- [Contributing](#contributing)
- [Who Built COVID Shield?](#who-built-covid-shield)

## Overview

_[Apple/Google's Exposure Notification](https://www.apple.com/covid19/contacttracing) specifications
provide important information to contextualize the rest of this document._

There are two fundamental operations conceptually:

* **Retrieving diagnosis keys**: retrieving a list of all keys uploaded by other users; and
* **Submitting diagnosis keys**: sharing keys returned from the EN framework with the server.

These two operations are implemented as two separate servers (`key-submission` and `key-retrieval`)
generated from this codebase, and can be deployed independently as long as they share a database. It
is also possible to deploy any number of configurations for each of these components, connected to
the same database, though there would be little value in deploying multiple configurations of
`key-retrieval`.

For a more technical overview of the codebase, especially of the protocol and database schema, see
[this video](https://www.youtube.com/watch?v=5GNJo1hEj5I).

### Retrieving diagnosis keys

When diagnosis keys are uploaded, the `key-submission` server stores the data defined and required
by the Exposure Notification API in addition to the time at which the data was received by the
server. This submission timestamp is rounded to the nearest hour for privacy preservation (to
prevent correlation of multiple keys to the same user).

The hour of submission is used to group keys into buckets, in order to prevent clients (the
soon-to-be-released _COVID Shield_ mobile app) from having to download a given set of key data
multiple times in order to repeatedly check for exposure.

The published diagnosis keys are fetched—with some best-effort authentication—from a Content
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

### Submitting diagnosis keys

In brief, upon receiving a positive diagnosis, a health care professional will generate a _One Time
Code_ through a web application frontend (a reference implementation will be open-sourced soon), which
communicates with `key-submission`. This code is sent to the patient, who enters the code into their
(soon-to-be-released) _COVID Shield_ App. This code is used to authenticate the
Application (once) to the diagnosis server. Encryption keypairs are exchanged by the Application
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
global diagnosis keys to each and every person in the world, but let's work through the number in
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

### Platforms

We hope to provide reference implementations on AWS, GCP, and Azure via [Hashicorp Terraform](https://www.terraform.io/).

[Amazon AWS](config/infrastructure/aws/README.md)

[Kubernetes](deploy/kubernetes/README.md)

## Metrics and Tracing

COVID Shield uses [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) to configure the metrics and tracing for the server, both the key retrieval and key submission.

### Metrics

Currently, the following options are supported for enabling Metrics:
* standard output
* prometheus

Metrics can be enabled by setting the `METRIC_PROVIDER` variable to `stdout`, `pretty`, or `prometheus`.

Both `stdout` and `pretty` will send metrics output to stdout but differ in their formatting. `stdout` will print
the metrics as JSON on a single line whereas `pretty` will format the JSON in a human-readable way, split across
multiple lines.

If you want to use Prometheus, please see the additional configuration requirements below.

#### Prometheus 

In order to use Prometheus as a metrics solution, you'll need to be running it in your environment. 

You can follow the instructions [here](https://prometheus.io/docs/prometheus/latest/installation/) for running Prometheus. 

You will need to edit the configuration file, `prometheus.yml` to add an additional target so it actually polls the metrics coming from the COVID Shield server:

```
...
    static_configs:
    - targets: ['localhost:9090', 'localhost:2222']
```

### Tracing 

Currently, the following options are supported for enabling Tracing:
* standard output

Tracing can be enabled by setting the `TRACER_PROVIDER` variable to `stdout` or `pretty`.

Both `stdout` and `pretty` will send trace output to stdout but differ in their formatting. `stdout` will print
the trace as JSON on a single line whereas `pretty` will format the JSON in a human-readable way, split across
multiple lines.

Note that logs are emitted to `stderr`, so with `stdout` mode, logs will be on `stderr` and metrics will be on `stdout`.

## Contributing

See the [_Contributing Guidelines_](CONTRIBUTING.md).

## Who Built COVID Shield?

COVID Shield was originally developed by [volunteers at Shopify](https://www.covidshield.app/). It was [released free of charge under a flexible open-source license](https://github.com/CovidShield/server).

This repository is being developed by the [Canadian Digital Service](https://digital.canada.ca/). We can be reached at <cds-snc@tbs-sct.gc.ca>.
