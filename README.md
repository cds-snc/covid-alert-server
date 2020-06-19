[La version française suit.](#---------------------------------------------------------------------)

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

[Technical overview](https://github.com/CovidShield/server#overview) 

## Contributing

See the [_Contributing Guidelines_](CONTRIBUTING.md).

## Who Built COVID Shield?

COVID Shield was originally developed by [volunteers at Shopify](https://www.covidshield.app/). It was [released free of charge under a flexible open-source license](https://github.com/CovidShield/server).

This repository is being developed by the [Canadian Digital Service](https://digital.canada.ca/). We can be reached at <cds-snc@tbs-sct.gc.ca>.

## ---------------------------------------------------------------------

# Serveur de diagnostic COVID Shield

![Version de conteneurs](https://github.com/CovidShield/server/workflows/Container%20Builds/badge.svg)

Adapté de <https://github.com/CovidShield/server> ([voir les changements](https://github.com/cds-snc/covid-shield-server/blob/master/FORK.md))

Ce dépôt met en œuvre un serveur de diagnostic à utiliser comme serveur pour le cadriciel [Notification d’exposition](https://www.apple.com/covid19/contacttracing) d’Apple et de Google, suivant l’[orientation fournie par les commissaires à la protection de la vie privée du Canada](https://priv.gc.ca/fr/nouvelles-du-commissariat/allocutions/2020/s-d_20200507/).

Les choix de mise en œuvre visent à maximiser la protection des renseignements personnels, la sécurité et le rendement. Aucun renseignement permettant d’identifier une personne n’est stocké, et uniquement une adresse IP est disponible sur le serveur. Aucune donnée n’est conservée après 21 jours. Ce serveur est conçu pour gérer une utilisation par un maximum de 38 millions de Canadiens, bien qu’il puisse être adapté à n’importe quelle taille de population.

[Aperçu technique](https://github.com/CovidShield/server#overview) 

## Contribution

Voir les [_Directives de contribution_](CONTRIBUTING.md).

## Qui a conçu COVID Shield?

COVID Shield a été conçu à l’origine par des [bénévoles de Shopify](https://www.covidshield.app/). Il a été [diffusé gratuitement en vertu d’une licence ouverte flexible](https://github.com/CovidShield/server).

Ce dépôt est élaboré par le [Service numérique canadien](https://numerique.canada.ca/). Vous pouvez nous joindre à <cds-snc@tbs-sct.gc.ca>.
