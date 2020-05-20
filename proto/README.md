# CovidShield Diagnosis Server Protocol
_(Valid for [Exposure Notification API](https://www.apple.com/covid19/contacttracing) v1.2)_

*See [this joint statement from Canada's Federal, Provincial, and Territorial Privacy
Commissioners](https://priv.gc.ca/en/opc-news/speeches/2020/s-d_20200507/) for background on the
privacy concerns addressed by this protocol.*

This repository contains the protocol definition for communication between the [COVID Shield
Diagnosis Server](https://github.com/CovidShield/server) and the soon-to-be-open-sourced COVID
Shield Mobile Applications.

Some of the protocol is defined by [Google Protocol
Buffers](https://developers.google.com/protocol-buffers). The file containing these definitions is
in this repository at [covidshield.proto](covidshield.proto).

The Diagnosis Server implements five main endpoints:


* `/retrieve-day`: Fetch a set of Diagnosis Keys for a given day
* `/retrieve-hour`: Fetch a set of Diagnosis Keys for a specific hour in a day
* `/upload`: Upload a batch of Diagnosis Keys
* `/new-key-claim`: Generate One-Time-Code to permit an app user to upload keys
* `/claim-key`: Convert One-Time-Code into a credential that permits upload

Because of the fairly divergent requirements in terms of the consumers of these various endpoints,
most of them use different protocols, documented below:

## `/new-key-claim`

The user posts to the URL with an empty request body and token authentication.
The response is an 8-digit plain-text UTF-8/ASCII-encoded numeric code.

#### Example Request:
    POST /new-key-claim
    Authorization: Bearer i-am-a-medical-professional-and-this-is-my-token

#### Example Response:
    Content-Type: text/plain; charset=utf-8

    12345678

Since you, the reader, are more likely to be in a position of having to implement a client for this
endpoint yourself, we've provided examples of this in a handful of languages at
[`examples/new-key-claim`](https://github.com/CovidShield/server/tree/master/examples/new-key-claim).

When implementing this, please be cautious with your authorization token: it shouldn't be sent to
the user's browser (i.e. please don't implement this workflow in client-side javascript).

## `/claim-key`

The user (app) posts to the endpoint with a serialized protobuf of type
[KeyClaimRequest](covidshield.proto). The server responds with a
[KeyClaimResponse](covidshield.proto). Additional documentation can be found attached to the
definitions in [covidshield.proto](covidshield.proto).

#### Example Request:
    POST /claim-key
    Content-Type: application/x-protobuf

    <serialized KeyClaimRequest>

#### Example Response:
    Content-Type: application/x-protobuf

    <serialized KeyClaimResponse>

## `/upload`

The user (app) posts to the endpoint with a serialized protobuf of type
[EncryptedUploadRequest](covidshield.proto). The server responds with an
[EncryptedUploadResponse](covidshield.proto). Additional documentation can be found attached to the
definitions in [covidshield.proto](covidshield.proto).

#### Example Request:
    POST /upload
    Content-Type: application/x-protobuf

    <serialized EncryptedUploadRequest>

#### Example Response:
    Content-Type: application/x-protobuf

    <serialized EncryptedUploadResponse>

Note: The client is expected to call this once on day T+0 when the user first receives their
positive diagnosis, and then should call it again each subsequent day, for days T+1 through T+13.
Duplicate keys will be filtered by the server. Some time on day T+14, the keypairs used for
encryption and authorization will become invalid and be purged.

## `/retrieve-day/:date/:hmac`

the hmac parameter in this case must be a hex-encoded SHA256 HMAC (64 characters) of:

    date + ":" + currentHour

where `date` is the ISO8601 datestamp from the URL (e.g. 2020-01-01), and `currentHour` is the
current UTC hour number (i.e. `floor(unixtime / 3600)`). `currentHour` must be agree with the server
to within +/- 1 hour in order for the request to be accepted.

Of course there's no reliable way to truly authenticate these requests in an environment where
millions of devices have immediate access to them upon downloading an Application: this scheme is
purely to make it much more difficult to casually scrape these keys.

Unlike the other endpoints, the retrieve endpoints don't return just a single serialized protobuf
message, but rather a stream of them, each length-prefixed with a big-endian 32-bit integer. This is
explained in a little more detail at the end of this document.

#### Example Request (request data for the entire UTC date 2000-01-01)
    GET /retrieve-day/2000-01-01/<hmac>

The date filter here corresponds to the time at which a Diagnosis Key was accepted by the Diagnosis
Server, NOT the date for which the TemporaryExposure/Diagnosis Keys being fetched were active.
However, the keys returned by this endpoint will only include key data from Keys active between 0
and 14 days ago (relative to the current time upon handling this request).

The rationale for this is: a client should fetch the key data for the past 14 days initially.
There's no need to cache historical keys locally, or to ever fetch them again or feed them into
future ExposureSessions, as long as the application has recorded locally that keys have been
fetched and checked for that range of historical time. This implementation is designed in a way that
a device can check hourly for newly-available unprocessed data packs and expect to find one new one
each hour to feed into an ExposureSession.

Note that, over time, historical packs will get smaller: the server will prune keys that, at the
time of pack generation, are more than 14 days old. However, no new keys will ever be added to a
historical pack, so there is no value in re-fetching old packs once they have been processed.

## `/retrieve-day/:date/:hour/:hmac`

the hmac parameter in this case must be a hex-encoded SHA256 HMAC (64 characters) of:

    date + ":" + hour + ":" + currentHour

where `date` is the ISO8601 datestamp from the URL (e.g. 2020-01-01), `hour` is the hour number from
the URL padded to a width of two characters (e.g. "02"), and `currentHour` is the current UTC hour
number (i.e. `floor(unixtime / 3600)`). `currentHour` must be agree with the server to within +/- 1 hour in
order for the request to be accepted.

#### Example Request (request data for the first hour of UTC date 2000-01-01)
    GET /retrieve-hour/2000-01-01/00/<hmac>

(note that valid values for `:hour` are 00–23, and must be padded to two digits)

This is essentially identical to /retrieve-day, except that it returns only a single hour's
contents. It will never return data for the current hour. This is useful for incremental checking
during the current day, rather than waiting for the full-day pack at 8pm EDT.

## How to Retrieve Keys

Since the Exposure Notification Framework doesn't track keys before it is enabled, and since the
Framework never allows extraction of a key that is still active, there is little value in retrieving
data from before the App was installed.

However, an implementor shouldn't assume that the device is always connected to the internet, and
a full history of 14 days of keys will be relevant if the App has been installed for over 14 days.
So, each time an Application checks for new keys, they should fetch every un-fetched pack in the
previous 14 days, and mark a pack as completed once the Exposure Session has run successfully.

`retrieve-hour` should be used for packs within the current UTC day, and `retrieve-day` should be
used for previous days if a device has gone multiple days without an Internet connection (or without
the app running, for any other reason). `retrieve-hour` will not work for times more than a day or
so ago.

Since there is little value in retrieving historical data upon Application installation, it is
recommended to mark the previous 366 (number of hours in 14 days) hours as having already been
fetched, immediately on first run.

An example sketch of this suggested implementation can be found at
[examples/retrieval/app.rb](https://github.com/CovidShield/server/blob/master/examples/retrieval/app.rb).

## Response format for /retrieve-*

These endpoints return a stream of serialized File messages, each prefixed with a big-endian uint32
indicating the length of the following serialized message in bytes.

Each File corresponds to a particular region, and a single region may have multiple Files in the
stream if the keys from that region exceed 500kB (a framework-defined limit) for that snapshot.

#### Example Response
    Content-Type: application/x-protobuf; delimited=true

    <length><serialized EncryptedUploadResponse><length><serialized EncryptedUploadResponse>

Or, if the EncryptedUploadResponse is only 5 bytes long somehow, and there's only one of them, you
might see `00 00 00 05 xx xx xx xx xx`

You can find a large-ish example of this format in this repository at
`build/retrieve-example.proto-stream` after running `make test`.

Note, as a special case, that if there are no keys at all in the requested range, the total content
of the response will be `00 00 00 00`; that is, a big-endian uint32 of zero.

## Who Built COVID Shield?

We are a group of Shopify volunteers who want to help to slow the spread of COVID-19 by offering our
skills and experience developing scalable, easy to use applications. We are releasing COVID Shield
free of charge and with a flexible open-source license.

For questions, we can be reached at <press@covidshield.app>.
