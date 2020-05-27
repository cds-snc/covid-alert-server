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


* `/retrieve`: Fetch a set of Diagnosis Keys for a given two-hour period
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

## `/retrieve/:region/:period/:hmac`

The `region` is an [MCC](https://www.mcc-mnc.com/) (e.g. "302" for Canada).

An "hour number" in this system is a UTC timestamp divided (using integer division) by 3600. This
quantity increases by 1 each hour.

A period is an hour number, rounded down to the next lower even number. So for example, the period
for hours 3 and 2 is 2, in both cases. The period increases by 2 every 2 hours.

the hmac parameter in this case must be a hex-encoded SHA256 HMAC (64 characters) of:

    region + ":" + period + ":" + currentHour

where `period` is provided in the URL (e.g. `441670`), and `currentHour` is the current UTC hour
number (e.g. `441683`). `currentHour` must agree with the server to within +/- 1 hour in order for
the request to be accepted.

Of course there's no reliable way to truly authenticate these requests in an environment where
millions of devices have immediate access to them upon downloading an Application: this scheme is
purely to make it much more difficult to casually scrape these keys.

#### Example Response
    Content-Type: application/zip
    Cache-Control: max-age=3600, max-stale=600

    <zip-file>

Unlike the other endpoints, the retrieve endpoint doesn't return just a single serialized protobuf
message, but rather a zip file containing two files: `encoded.bin` contains a serialized
`TemporaryKeyExport`, and `encoded.sig` contains a serialized `TEKSignatureList`. These are passed
as-is to the Exposure Notification Framework.

Note that the `period` provided to the retrieve endpoint corresponds to the time at which a
Diagnosis Key was accepted by the Diagnosis Server, NOT the date for which the
TemporaryExposure/Diagnosis Keys being fetched were active. However, the keys returned by this
endpoint will only include key data from Keys active between 0 and 14 days ago (relative to the
current time upon handling this request).

The rationale for this is: a client should fetch the key data for the past 14 days initially.
There's no need to cache historical keys locally, or to ever fetch them again or feed them into
future ExposureSessions, as long as the application has recorded locally that keys have been fetched
and checked for that range of historical time. This implementation is designed in a way that a
device can check every two hours for newly-available unprocessed data packs and expect to find one
new one each two hours to feed into an ExposureSession.

Note that, over time, historical packs will get smaller: the server will prune keys that, at the
time of pack generation, are more than 14 days old. However, no new keys will ever be added to a
historical pack, so there is no value in re-fetching old packs once they have been processed.

Since the Exposure Notification Framework doesn't track keys before it is enabled, and since the
Framework never allows extraction of a key that is still active, there is little value in retrieving
data from before the App was installed.

However, an implementor shouldn't assume that the device is always connected to the internet, and a
full history of 14 days of keys (168 periods) will be relevant if the App has been installed for
over 14 days. So, each time an Application checks for new keys, they should fetch every un-fetched
pack in the previous 14 days, and mark a pack as completed once the Exposure Session has run
successfully.

Since there is little value in retrieving historical data upon Application installation, it is
recommended to mark the previous 168 periods as having already been fetched, immediately on first
run.

An example sketch of this suggested implementation can be found at
[examples/retrieval/app.rb](https://github.com/CovidShield/server/blob/master/examples/retrieval/app.rb).

## Who Built COVID Shield?

We are a group of Shopify volunteers who want to help to slow the spread of COVID-19 by offering our
skills and experience developing scalable, easy to use applications. We are releasing COVID Shield
free of charge and with a flexible open-source license.

For questions, we can be reached at <press@covidshield.app>.
