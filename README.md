# Net-Companion Lite

Outil de diagnostic réseau de terrain **portable « Zero-Install »** : un **binaire
unique** (Go + frontend Vue embarqué via `go:embed`) qui, au lancement, démarre
un serveur local sur `127.0.0.1:8080` et ouvre le navigateur par défaut.

Destiné aux ingénieurs NetOps intervenant physiquement sur site : résultats
visuels immédiats, aucune installation ni fichier de configuration à manipuler.

## Fonctionnalités

- **Port-Finder** — localise physiquement le poste via SNMP (BRIDGE-MIB) :
  switch, port (ex. `GigabitEthernet1/0/5`) et VLAN, en langage naturel.
- **Radar** — topologie de niveau 2 à partir de la table ARP (vérité L2),
  réchauffée par un sweep concurrent du sous-réseau ; graphe interactif
  (vis-network) avec fabricant résolu via base **OUI IEEE embarquée** (~40 000
  fabricants).
- **In-App Vault** — trousseau chiffré **AES-256-GCM**, clé dérivée d'un **PIN**
  (Argon2id). Identifiants SNMP/SSH stockés dans `%APPDATA%/netcompanion/vault.dat`
  (jamais en clair), déverrouillés en RAM.
- **Config-Diff** — récupère `running-config` vs `startup-config` via SSH et
  affiche un diff coloré (vert = ajout, rouge = suppression).
- **Contournement NAC** :
  - *Niveau 1* — écoute passive **LLDP/CDP** (switch/port même sans IP).
  - *Niveau 2* — **MAC Spoofing** intégré (aperçu + application avec élévation).

## Prérequis (développement)

- Go 1.22+ (testé avec 1.27)
- Node.js 18+ / npm

## Build

Un seul script produit le binaire (front Vite → `go:embed` → exécutable) :

```powershell
# Windows
pwsh -File build.ps1
```

```bash
# Linux / macOS
./build.sh
```

Résultat : `backend/net-companion.exe` (ou `backend/net-companion`).

## Lancement

Double-cliquez le binaire, ou :

```powershell
backend\net-companion.exe
```

Le navigateur s'ouvre sur `http://127.0.0.1:8080`. Au premier lancement,
créez un **PIN** ; aux suivants, saisissez-le pour déverrouiller le trousseau.

Variable d'environnement optionnelle : `NC_ADDR` pour changer l'adresse
d'écoute (par défaut `127.0.0.1:8080`).

## Utilisation

1. Créez/déverrouillez le coffre (PIN).
2. Dans le drawer **Coffre**, ajoutez vos communities SNMP et identifiants SSH.
3. **Radar** → *Rescanner* pour peupler le graphe de topologie.
4. **Localiser mon port** (bandeau) → interroge le switch via SNMP.
5. **Config-Diff** → saisissez l'IP d'un équipement pour comparer running/startup.
6. **NAC** → écoute LLDP, ou aperçu/application d'un MAC spoof.

## Limites connues (MVP)

- **Capture LLDP réelle** : nécessite **Npcap** installé et une compilation
  `go build -tags npcap` (+ compilateur C). Sans cela, l'écoute LLDP renvoie
  proprement « indisponible » (dégradation gérée).
- **MAC Spoofing réel** : nécessite une session **administrateur/root** ; sans
  élévation, l'action est refusée proprement (aperçu du plan disponible).
- **Radar** : sous Windows sans admin, l'ICMP retombe sur TCP + ARP. Certains
  points d'accès Wi-Fi acceptant tout le trafic TCP, la topologie s'appuie sur
  la **table ARP** (fiable) plutôt que sur les résultats bruts du sweep.
- **Port-Finder SNMP** : logique validée par tests mockés ; le chemin réseau
  réel dépend d'un équipement SNMP joignable.

## Sécurité

- **Coffre** chiffré AES-256-GCM, clé dérivée du PIN (Argon2id) ; secrets en RAM
  seulement une fois déverrouillé.
- **API locale protégée** : au démarrage, un **jeton de session** aléatoire est
  généré et injecté dans la page servie. Chaque appel `/api/*` exige l'en-tête
  `X-NC-Token` + une **Origin** locale. Un site web tiers ouvert dans le
  navigateur ne peut donc pas atteindre l'API (protection CSRF / DNS-rebinding),
  car il ne peut ni lire le jeton (CORS) ni poser l'en-tête custom.
- **Limite assumée** : sur l'interface loopback, un *autre processus local*
  disposant des droits de l'utilisateur peut lire la page (et donc le jeton).
  L'isolation inter-processus locale dépasse le cadre d'un binaire portable ;
  la menace visée ici est le navigateur.

## Signature du binaire

Le binaire n'est pas signé par défaut → Windows SmartScreen / macOS Gatekeeper
peuvent alerter. Pour une diffusion, signer :

- **Windows** : `signtool sign /fd SHA256 /a /tr http://timestamp.digicert.com /td SHA256 net-companion.exe`
  (ou `osslsigncode` avec un certificat Authenticode).
- **macOS** : `codesign --deep --options runtime --sign "Developer ID Application: …" net-companion`
  puis notarisation (`notarytool`).
- **Linux** : signer le paquet/dépôt (GPG) selon la distribution.

## Architecture

```
frontend/           # Vue 3 + Vite (buildé vers backend/web/dist)
backend/
  main.go           # serveur + ouverture navigateur
  web/embed.go      # //go:embed dist
  internal/
    server/         # routeur HTTP + statique
    vault/          # AES-256-GCM + Argon2id
    api/            # handlers REST (/api/*)
    network/
      netinfo/ arp/ radar/ oui/ portfinder/ configdiff/ lldp/ macspoof/
    models/         # DTO JSON partagés
```

## Tests

```bash
cd backend && go test ./...
```
