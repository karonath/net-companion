# Net-Companion **Lite**

**Le couteau suisse réseau du technicien de terrain, sur une clé USB.**

Un binaire unique, portable et sans installation : copié sur une clé USB, lancé
sur n'importe quel poste, il ouvre son interface dans le navigateur et fournit
en quelques secondes un diagnostic réseau complet, visuel et exportable.

*Auteur : **Charlys Menuet** (Karonath).*

---

# Partie 1 — Métier

## Le problème

Sur site, l'ingénieur réseau (NetOps) perd du temps à répéter les mêmes gestes
avec une multitude d'outils disparates : trouver sur quel port d'un switch il
est branché, savoir qui est présent sur le réseau, vérifier que la connectivité
est saine, comparer une configuration, produire un compte-rendu. Le tout
souvent sans droits d'installation sur le poste client, et parfois sans même
obtenir d'adresse IP.

## La proposition de valeur

Net-Companion Lite rassemble ces gestes dans **un seul outil portable**,
**sans installation** ni configuration, qui donne un **résultat visuel immédiat**
et un **livrable prêt à joindre à un ticket**. Le technicien branche sa clé,
lance l'outil, et diagnostique.

## Pour qui

- Ingénieurs et techniciens **NetOps / réseau** en intervention sur site.
- **Prestataires et infogérants** qui doivent documenter chaque passage.
- **Support N2/N3** ayant besoin d'un état des lieux rapide et fiable.

## Ce que l'outil apporte, fonctionnalité par fonctionnalité

| Fonction | Bénéfice métier | Prérequis |
|---|---|---|
| **Check de site (1 clic)** | Un état des lieux complet du site en quelques secondes, horodaté et **exportable** (rapport imprimable / JSON) à joindre au ticket ou remettre au client. | — |
| **Radar / topologie** | Visualiser instantanément qui est présent sur le réseau, avec le **fabricant** de chaque équipement ; cliquer un hôte pour ses détails et agir dessus. | — |
| **Diagnostics** | Répondre en 1 clic à « le réseau fonctionne-t-il ? » : passerelle, DNS, accès Internet, latence ; plus test de port et traceroute. | — |
| **Port-Finder** | Savoir **où l'on est physiquement branché** : switch, port et VLAN, en langage clair. Fini la chasse au port dans l'armoire. | Identifiant SNMP |
| **Voisinage (LLDP/CDP)** | Cartographier les liens **switch-à-switch** réels pour comprendre la topologie. | Identifiant SNMP |
| **Config-Diff** | Repérer immédiatement les modifications de configuration **non sauvegardées** d'un équipement. | Identifiant SSH |
| **Configs & dérive** | Sauvegarder la configuration de **plusieurs équipements** et détecter toute **dérive** vis-à-vis d'une référence approuvée. | Identifiant SSH |
| **Contournement de blocage (NAC)** | Rester opérationnel quand la prise est verrouillée : usurpation d'adresse MAC et identification passive du port. | Selon le cas |
| **Coffre-fort d'identifiants** | Transporter en sécurité les secrets d'entreprise (SNMP/SSH), chiffrés et protégés par un code PIN, réutilisés automatiquement. | — |

## Prise en main (sans matériel)

1. Copier le binaire sur une clé USB, le lancer → l'interface s'ouvre dans le
   navigateur.
2. Créer un **code PIN** (il chiffre les identifiants sur le poste).
3. Cliquer **« Mode démo »** : un équipement réseau **simulé** démarre — toutes
   les fonctions se testent **sans aucun matériel**.
4. Sur chaque écran, l'aide **« À quoi ça sert ? »** explique la fonctionnalité.

Sans matériel ni mode démo, le **Radar**, les **Diagnostics** et le **Check de
site** fonctionnent directement sur le réseau du poste.

---

# Partie 2 — Technologie

## Principes

- **Binaire unique, zéro-install** : le frontend est compilé puis embarqué dans
  l'exécutable (`go:embed`). Rien à déployer, rien à configurer.
- **Local et privé** : serveur sur `127.0.0.1:8080`, ouverture automatique du
  navigateur ; aucune donnée ne quitte le poste.
- **Dégradation propre** : une capacité indisponible (droits, dépendance
  absente) est signalée clairement, jamais bloquante.

## Pile technique

- **Backend** : Go (1.22+). Réseau : SNMP (v2c/v3), SSH, ARP, ICMP/TCP,
  LLDP/CDP-MIB. Base **OUI IEEE** embarquée pour l'identification des fabricants.
- **Frontend** : Vue 3 (Composition API) + Vite, graphe de topologie
  `vis-network`, entièrement embarqué (zéro CDN).
- **Packaging** : un seul exécutable Windows / Linux / macOS.

## Sécurité

- **Coffre** : clé dérivée du PIN par **Argon2id**, secrets chiffrés en
  **AES-256-GCM**, déchiffrés en mémoire uniquement
  (`%APPDATA%/netcompanion/vault.dat`).
- **API locale protégée** : jeton de session + contrôle d'origine, pour empêcher
  qu'un site web tiers ouvert dans le navigateur n'atteigne l'outil
  (anti CSRF / DNS-rebinding).
- **Limite assumée** : sur l'interface loopback, un autre processus local
  disposant des droits de l'utilisateur peut lire la page ; l'isolation
  inter-processus dépasse le cadre d'un binaire portable.

## Compilation

Prérequis : Go 1.22+ et Node.js 18+.

```powershell
pwsh -File build.ps1      # Windows
```
```bash
./build.sh                # Linux / macOS
```

Le script compile le frontend puis produit **un seul fichier**
`backend/net-companion(.exe)`. Variable optionnelle `NC_ADDR` pour changer
l'adresse d'écoute (défaut `127.0.0.1:8080`).

> Le binaire n'est pas signé : SmartScreen / Gatekeeper peuvent alerter au
> premier lancement. Pour une diffusion, signer avec `signtool` (Windows),
> `codesign` + notarisation (macOS) ou GPG (Linux).

## Architecture

```
frontend/           Vue 3 + Vite (build → backend/web/dist)
backend/
  main.go           serveur local + ouverture navigateur + jeton de session
  web/embed.go      frontend embarqué (go:embed)
  internal/
    server/         routeur HTTP + authentification + fichiers statiques
    vault/          coffre AES-256-GCM + Argon2id
    api/            API REST (/api/*)
    sim/            équipement simulé (SSH réel + SNMP de démo)
    history/        états de site + rapport
    configstore/    configurations par équipement + baseline/dérive
    network/        netinfo, arp, radar, oui, portfinder, configdiff,
                    lldp, macspoof, diag, neighbors
    models/         structures JSON partagées
```

## Tests

```bash
cd backend && go test ./...
```

---

# Partie 3 — Roadmap

## Édition terrain (nano-ordinateur + téléphone)

La prochaine étape n'est plus une clé USB mais un **nano-ordinateur** (type
Raspberry Pi) que le technicien pose sur site et branche au réseau. Le boîtier :

- fonctionne en autonomie et **capture passivement les trames** (LLDP/CDP), ce
  qui permet d'identifier switch et port **même quand le réseau ne délivre
  aucune IP** ;
- **héberge l'interface** ; le technicien la pilote **depuis son téléphone**
  (application mobile ou simple navigateur), idéalement via un point d'accès
  Wi-Fi diffusé par le boîtier.

**Toujours sans installation** — mais côté technicien : rien à installer sur le
téléphone, rien à déployer sur le réseau du client. C'est le matériel qui porte
l'outil ; le téléphone n'est qu'un écran de contrôle.

Techniquement, la capture passive s'active par un build dédié (`-tags npcap`) :
capture native via **libpcap** sur le nano-ordinateur **Linux**, ou **Npcap**
(+ SDK, + compilateur C) sous Windows. Ce mode dépend d'une bibliothèque de
capture : il est donc réservé au matériel de terrain, tandis que la version
**Lite** reste zéro-install et s'appuie sur le voisinage **par SNMP**.

## Pistes ultérieures

- **Application mobile** dédiée (habillage de l'interface, appairage au boîtier
  par QR code).
- **Signature et distribution** du binaire (suppression des alertes SmartScreen).
- **Validation sur matériel réel** (SNMPv3, LLDP/CDP, SSH) en complément du
  simulateur.
- **Multi-sites / profils** clients avec historique et baselines dédiés.
- **Intégrations** : export vers ticketing, Slack/Teams, ou une supervision.

---

*Net-Companion — Charlys Menuet (Karonath).*
