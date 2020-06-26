[La version française suit.](#serveur-de-diagnostic-covid-shield)

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

____

# Serveur de diagnostic COVID Shield

![Container Build](https://github.com/CovidShield/server/workflows/Container%20Builds/badge.svg)

Adapté à partir de <https://github.com/CovidShield/server> ([voir les modifications](https://github.com/cds-snc/covid-shield-server/blob/master/FORK.md))

Ce dépôt implémente un serveur de diagnostic à utiliser comme serveur pour le [cadriciel de notification d’exposition](https://www.apple.com/covid19/contacttracing) d’Apple et de Google, suivant les [directives fournies par les commissaires à la protection de la vie privée du Canada](https://priv.gc.ca/fr/nouvelles-du-commissariat/allocutions/2020/s-d_20200507/).

Les choix faits dans l’implémentation visent à maximiser la confidentialité, la sécurité et le rendement. Les renseignements identificatoires ne sont jamais stockés, et il n’y a que l’adresse IP qui est accessible au serveur. Aucune donnée n’est conservée après 21 jours. Ce serveur est conçu pour gérer jusqu’à 38 millions d’utilisateurs canadiens, même s’il peut être étendu à n’importe quelle taille de population.

Dans la présente documentation :

- [Aperçu](#aperçu)
   - [Récupération des clés de diagnostic](#récupération-des-clés-de-diagnostic)
   - [Récupération de la configuration d’exposition](#récupération-de-la-configuration-de-lexposition)
   - [Envoi des clés de diagnostic](#envoyer-les-clés-de-diagnostic)
- [Utilisation des données](#utilisation-des-données)
- [Génération de codes uniques](#génération-de-codes-à-usage-unique)
- [Documentation du protocole](#documentation-du-protocole)
- [Remarques de déploiement](#remarques-de-déploiement)
- [Indicateurs et traçage](#indicateurs-et-traçage)
- [Contribution](#contribution)
- [Qui a conçu COVID Shield?](#qui-a-conçu-covid-shield)

## Aperçu

_Les [spécifications de la notification d’exposition d’Apple et de Google](https://www.apple.com/covid19/contacttracing) fournissent des renseignements importants pour contextualiser le reste de ce document._

Il y a deux opérations fondamentales sur le plan conceptuel :

* **Récupération des clés de diagnostic** : récupération d’une liste de toutes les clés téléversées par d’autres utilisateurs;
* **Envoi des clés de diagnostic** : partage des clés renvoyées par le cadriciel de notification d’exposition avec le serveur.

Ces deux opérations sont implémentées en tant que deux serveurs distincts (`key-submission` et `key-retrieval`) générés à partir de cette base de code, et peuvent être déployées indépendamment tant qu’elles partagent une base de données. Il est également possible de déployer n’importe quel nombre de configurations pour chacun de ces composants, connectés à la même base de données, même s’il y aurait peu d’utilité à déployer plusieurs configurations de `key-retrieval`.

Pour une vue d’ensemble technique du code de base, particulièrement du protocole et du schéma de base de données, voir [cette vidéo](https://www.youtube.com/watch?v=5GNJo1hEj5I).

### Récupération des clés de diagnostic

Au moment du téléversement des clés de diagnostic, le serveur `key-submission` stocke les données définies et requises par l’interface de programmation d’applications (API) de notification d’exposition en plus de la date à laquelle les données ont été reçues par le serveur. L’horodatage de cet envoi est arrondi à l’heure la plus proche pour la protection de la vie privée (pour empêcher la corrélation de plusieurs clés avec le même utilisateur).

L’heure d’envoi est utilisée pour regrouper les clés en compartiments, afin d’empêcher que les clients (la sortie prochaine de l’application mobile _COVID Shield_) aient à télécharger un certain ensemble de données de clés plusieurs fois pour pouvoir vérifier l’exposition de manière répétée.

Les clés de diagnostic publiées sont extraites (avec une authentification optimisée) à partir d’un réseau de distribution du contenu (RDC), soutenu par `key-retrieval`. Cela permet un nombre fonctionnellement arbitraire d’utilisateurs simultanés.

### Récupération de la _configuration de l’exposition_

[_La configuration de l’exposition_](https://developer.apple.com/documentation/exposurenotification/enexposureconfiguration), utilisée pour déterminer le risque d’une exposition donnée, est également récupérée sur le serveur `key-retrieval`. Un document JSON décrivant la configuration d’exposition actuelle pour une région donnée est disponible par le chemin `/exposure-configuration/<region>.json`, par exemple pour l’Ontario (région `ON`) :

```sh
$ curl https://retrieval.covidshield.app/exposure-configuration/ON.json
{"minimumRiskScore":0,"attenuationLevelValues":[1,2,3,4,5,6,7,8],"attenuationWeight":50,"daysSinceLastExposureLevelValues":[1,2,3,4,5,6,7,8],"daysSinceLastExposureWeight":50,"durationLevelValues":[1,2,3,4,5,6,7,8],"durationWeight":50,"transmissionRiskLevelValues":[1,2,3,4,5,6,7,8],"transmissionRiskWeight":50}
```

### Envoyer les clés de diagnostic

En bref, lorsque qu’un diagnostic positif est établi, le professionnel de la santé générera un _code à usage unique_ avec une application Web frontale (une implémentation de référence sera bientôt disponible en code source libre) qui communique avec `key-submission`. Ce code est envoyé au patient, qui entre le code dans son application _COVID Shield_ (bientôt disponible). Ce code est utilisé pour authentifier l’application (une fois) vis-à-vis le serveur de diagnostic. Les paires de clés de chiffrement sont échangées par l’application et le serveur `key-submission` et sont stockée pendant quatorze jours, et la base de données est immédiatement purgée du code à usage unique.

Ces paires de clés sont utilisées pour chiffrer et autoriser les téléversements de _clé de diagnostic_ pendant les quatorze jours qui suivent, après quoi elles sont enlevées de la base de données.

Le schéma de chiffrement utilisé pour le téléchargement de clés est _NaCl Box_ (un schéma de chiffrement de clé publique utilisant Curve25519, XSalsa20 et Poly1305). Il s’agit d’une implémentation considérée extrêmement sécuritaire de la cryptographie à courbe elliptique.

## Utilisation des données

Le protocole de récupération des _clés de diagnostic_ utilisé dans _COVID Shield_ a été conçu pour limiter le transfert de données à un minimum. Considérant le grand nombre de clés, et en supposant que le client les extraie en utilisant la compression, il y a un surdébit de protocole minimal en plus de la taille des données de clé de 16 octets.

Dans tous les exemples ci-dessous :

* Chaque cas peut générer jusqu’à 28 clés.
* Les clés sont valides et distribuées pendant 14 jours.
* Chaque clé implique un peu moins de 18 octets de transfert de données pendant l’utilisation de la compression.
* Les métadonnées et le surdébit de protocole des clés devraient en réalité être minimes, mais : 
* Supposez que les nombres sont 50 % plus élevés que ce qui se trouve ci-dessous pour plus de sûreté. Ce fichier Readme sera mis à jour bientôt avec des tailles de données réelles plus précises.

**Les données ci-dessous datent du 12 mai 2020**. Pour chaque cas, nous supposons que les exemples de nouveaux cas recensés sont une récurrence quotidienne constante.

### Déployé uniquement dans la province d’Ontario

Il y a eu 350 nouveaux cas en Ontario le 10 mai 2020 : 350 * 28 * 18 = 170 ko par jour. Ainsi, un déploiement dans la province de l’Ontario au taux d’infection actuel engendrerait **7,1 ko de téléchargement par heure**.

### Déployé au Canada

Le 10 mai 2020, il y a eu 1100 nouveaux cas au Canada : 1100 * 28 * 18 = 540 ko par jour. Ainsi, le déploiement au Canada au taux d’infection actuel entraînerait **23 ko de téléchargement par heure**.

### Déployé dans l’ensemble des États-Unis d’Amérique

Il y a eu 18 000 nouveaux cas aux États-Unis le 10 mai 2020 : 18 000 * 28 * 18 = 8,9 mégaoctets [Mo] par jour. Ainsi, le déploiement dans l’ensemble des États-Unis au taux d’infection actuel entraînerait **370 ko de téléchargement par heure**.

### Déployé dans le monde entier

Si _COVID Shield_ était déployé dans le monde entier, nous serions enclins à utiliser les « régions » conçues dans le protocole pour établir des espaces de noms pour les clés, afin de ne pas desservir l’ensemble des clés de diagnostic mondiales pour chaque personne dans le monde. Passons cependant en revue les chiffres au cas où nous ne le ferions pas : 

Le 10 mai 2020, il y a eu 74 000 nouveaux cas dans le monde : 74 000 * 28 * 16 = 36 Mo par jour. Ainsi, le déploiement dans le monde entier au taux d’infection actuel entraînerait **1,5 Mo de téléchargement par heure**.

## Génération de codes à usage unique

Nous utilisons un système de génération de codes à usage unique qui permet aux professionnels authentifiés d’émettre des codes. Ces codes doivent être transmis aux patients présentant un diagnostic positif par l’intermédiaire de n’importe quel canal de communication pratique.

Cette démarche dépend d’un service différent, qui détient des justificatifs pour communiquer avec ce serveur (`key-submission`).
Nous avons une implémentation à titre d’exemple dont le code source sera bientôt ouvert. Cependant, nous nous attendons à ce que les autorités sanitaires préfèrent intégrer cette fonctionnalité dans leurs systèmes existants. L’intégration est extrêmement simple, et on dispose [d’exemples en plusieurs languages](https://github.com/CovidShield/server/tree/master/examples/new-key-claim). Au minimum :

```bash
curl -XPOST -H "Authorization: Bearer $token" "https://submission.covidshield.app/new-key-claim"
```

## Documentation du protocole

Pour une description détaillée du protocole, veuillez consulter [le sous-répertoire « proto » de ce dépôt](/proto).

## Remarques de déploiement

- `key-submission` dépend du déploiement derrière un pare-feu (par exemple [AWS WAF](https://aws.amazon.com/waf/), ce qui freine les utilisateurs de manière agressive par des réponses 400 et 401.

- `key-retrieval` suppose un déploiement derrière un proxy inverse de mise en cache.

### Plateformes

Nous espérons fournir des implémentations de référence sur AWS, GCP et Azure par [Hashicorp Terraform](https://www.terraform.io/).

[Amazon AWS](config/infrastructure/aws/README.md)

[Kubernetes](deploy/kubernetes/README.md)

## Indicateurs et traçage

COVID Shield utilise [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) pour configurer les indicateurs et le traçage du serveur, à la fois pour la récupération et l’envoi des clés.

### Indicateurs 

Actuellement, les options suivantes sont prises en charge pour activer les indicateurs :
* données de sortie standard
* prometheus

Les indicateurs peuvent être activés en définissant la variable `METRIC_PROVIDER` sur `stdout`, `pretty`, ou `prometheus`.

Aussi bien `stdout` que `pretty enverront les indicateurs de sortie à `stdout`, mais leur mise en forme diffère. `stdout` imprimera les indicateurs en tant que JSON sur une seule ligne, tandis que `pretty` formatera le JSON de manière lisible pour les humains, avec une séparation sur plusieurs lignes.

Si vous voulez utiliser Prometheus, veuillez consulter les exigences de configuration supplémentaires ci-dessous.

#### Prometheus 

Pour utiliser Prometheus comme solution d’indicateurs, vous devez l’exécuter dans votre environnement. 

Vous pouvez suivre les instructions [ici](https://prometheus.io/docs/prometheus/latest/installation/) pour exécuter Prometheus. 

Vous devrez éditer le fichier de configuration `prometheus.yml` pour ajouter une cible supplémentaire afin qu’il interroge réellement les indicateurs provenant du serveur COVID Shield :

```
...
    static_configs:
    - targets: ['localhost:9090', 'localhost:2222']
```

### Traçage  

Actuellement, les options suivantes sont prises en charge pour activer le traçage :
* données de sortie standard

Le traçage peut être activé en définissant la variable `TRACER_PROVIDER` sur `stdout` ou `pretty`.

Aussi bien `stdout` que `pretty` enverront le traçage de sortie à `stdout`, mais leur mise en forme diffère. `stdout` imprimera le traçage en tant que JSON sur une seule ligne, tandis que `pretty` formatera le JSON de manière lisible pour les humains, avec une séparation sur plusieurs lignes.

Notez que les journaux sont émis en mode `stderr`, de sorte qu’avec le mode `stdout`, les journaux seront en mode `stderr` et les indicateurs seront en mode `stdout`.

## Contribution

Consultez les [_Directives de contribution_](CONTRIBUTING.md).

## Qui a conçu COVID Shield?

COVID Shield a été développé à l’origine par [des bénévoles de Shopify](https://www.covidshield.app/). Il a été [diffusé gratuitement en vertu d’une licence ouverte flexible](https://github.com/CovidShield/server).

Ce dépôt est maintenu par le [Service numérique canadien](https://numerique.canada.ca/). Vous pouvez nous joindre à <cds-snc@tbs-sct.gc.ca>.
