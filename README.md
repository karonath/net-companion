# Net-Companion **Lite**

**Le couteau suisse réseau du technicien de terrain, sur une clé USB.**

Un **binaire unique, zéro-install** (Go + frontend Vue embarqué) : tu le copies
sur une clé USB, tu le lances sur n'importe quelle machine, il ouvre son
interface dans le navigateur sur `http://127.0.0.1:8080`. Aucune installation,
aucune dépendance, aucun fichier de config à trimballer.

> **Deux éditions**
> - **Lite (cette version)** — pensée pour la **clé USB** : portable, zéro-install,
>   tout en RAM/loopback. C'est la version que tu tiens.
> - **Édition terrain (à venir)** — pensée pour un **nano-ordinateur** posé sur
>   site, qui ajoute la **capture passive de trames** (LLDP/CDP sans IP) via
>   Npcap + un build `-tags npcap`. Non portable (dépend de Npcap), mais plus
>   puissante quand le réseau te bloque totalement. Voir *Édition terrain* plus bas.

---

## Démarrage en 30 secondes (DevOps qui découvre)

1. Copie `net-companion.exe` sur ta clé, lance-le → le navigateur s'ouvre.
2. **Crée un code PIN** (il chiffre tes identifiants sur la machine).
3. Clique **« Mode démo »** dans le bandeau du haut → un équipement réseau
   **simulé** démarre : tu peux **tout essayer sans aucun matériel**.
4. Sur chaque panneau, déplie **« À quoi ça sert ? »** pour comprendre la feature.

Pas de matériel ? Le **Radar**, les **Diagnostics** et le **Check de site**
fonctionnent quand même sur ton propre réseau. Le reste (Port-Finder,
Voisins, Config-Diff, Configs) s'essaie via le **Mode démo**.

---

## Fonctionnalités & utilité

| Fonction | Ce que ça t'apporte sur le terrain | Prérequis |
|---|---|---|
| **Check de site (1 clic)** | État des lieux complet en < 15 s (hôtes + diagnostics), horodaté, comparable d'un passage à l'autre, **rapport exportable** (HTML imprimable / JSON) à joindre au ticket. | — |
| **Radar / topologie L2** | Voir qui est présent sur le sous-réseau (table ARP + sondes), avec **fabricant** (base OUI embarquée). Clique un hôte → IP, MAC, nom d'hôte, latence + actions. | — |
| **Diagnostics** | « Le réseau marche-t-il ? » : passerelle, DNS, Internet, latence/jitter + **test de port** et **traceroute** à la demande. | — |
| **Port-Finder** | « Où suis-je physiquement branché ? » → **switch, port (ex. Gi1/0/5) et VLAN** en langage clair, via SNMP. | SNMP (coffre) ou Mode démo |
| **Voisins LLDP/CDP** | La vraie carte de proximité : « ce switch voit tel switch sur tel port », via SNMP. | SNMP (coffre) ou Mode démo |
| **Config-Diff** | Repérer les changements non sauvegardés : **running vs startup** d'un équipement (SSH), en diff coloré. | SSH (coffre) ou Mode démo |
| **Configs & drift** | Backup de la running-config de **plusieurs équipements**, **baseline approuvée** et détection de **dérive** vs cette référence. | SSH (coffre) ou Mode démo |
| **Coffre (Vault)** | Trousseau **chiffré** (PIN) : communities **SNMP v2c/v3** et identifiants **SSH**, réutilisés automatiquement. Bouton **Tester** un credential contre une IP. | — |
| **Blocage NAC** | Que faire quand la prise est verrouillée : **MAC spoofing** (usurper un appareil légitime) et **écoute passive LLDP** (identifier switch/port même sans IP). | MAC spoof : admin · Écoute passive : *édition terrain* |
| **Mode démo** | Tester **sans matériel** : démarre un switch simulé (SSH + SNMP) pour Port-Finder, Voisins, Config-Diff, Configs. | — |

---

## Sécurité

- **Coffre** : clé dérivée du PIN (Argon2id), secrets chiffrés **AES-256-GCM**,
  déchiffrés en RAM seulement. Fichier : `%APPDATA%/netcompanion/vault.dat`.
- **API locale protégée** : jeton de session injecté dans la page + en-tête
  `X-NC-Token` + contrôle d'Origin → un site web tiers ouvert dans ton
  navigateur ne peut pas atteindre l'API (anti CSRF / DNS-rebinding).
- **Limite assumée** : sur loopback, un autre processus local avec tes droits
  peut lire la page ; l'isolation inter-processus dépasse un binaire portable.
- Historique/configs = inventaire (pas de secrets) → JSON local en clair.

---

## Build (développeur)

Prérequis : Go 1.22+ et Node.js 18+.

```powershell
pwsh -File build.ps1      # Windows
```
```bash
./build.sh                # Linux / macOS
```
Produit `backend/net-companion.exe` (ou `net-companion`) : **un seul fichier**
(frontend Vite compilé puis embarqué via `go:embed`).

Lancement : double-clic, ou `backend\net-companion.exe`.
Variable optionnelle : `NC_ADDR` pour changer l'adresse d'écoute (défaut
`127.0.0.1:8080`).

> **Signature** : le binaire n'est pas signé → SmartScreen/Gatekeeper peuvent
> alerter au 1er lancement (« Informations complémentaires → Exécuter quand
> même »). Pour diffuser : `signtool` (Windows), `codesign`+notarisation
> (macOS), GPG (Linux).

---

## Édition terrain (nano-ordinateur, à venir)

L'**écoute passive LLDP/CDP** capte les trames du switch **même sans adresse IP**
(réseau/NAC totalement bloqué) — utile posté à demeure sur un nano-ordinateur.
Elle repose sur de la capture de paquets (gopacket/pcap = cgo) et exige donc :

1. **Npcap** installé sur la machine (npcap.com) ;
2. le **SDK Npcap** (en-têtes + lib) au moment du build ;
3. un **compilateur C** (`CGO_ENABLED=1`, mingw-w64/MSYS2) ;
4. la compilation avec le tag : `go build -tags npcap`.

Ce binaire n'est alors **plus zéro-install** (il dépend de Npcap sur chaque
cible) — c'est pourquoi il est réservé à l'édition terrain. La version **Lite**
par défaut désactive proprement cette capture (« indisponible ») et privilégie
le voisinage **par SNMP** (onglet Voisins), qui, lui, fonctionne sans rien.

---

## Limites connues (Lite)

- **Écoute passive LLDP** : désactivée (voir Édition terrain).
- **MAC spoofing réel** : nécessite une session administrateur ; sinon l'aperçu
  du plan est affiché mais l'action est refusée proprement.
- **Radar** : sans admin, s'appuie sur la table ARP (fiable en L2) plutôt que
  sur un ping ICMP.
- **Contre du vrai matériel** : Port-Finder / Voisins / Config-Diff / Configs
  nécessitent l'équipement joignable + des credentials valides. La logique est
  validée par tests et démontrable via le **Mode démo**.

---

## Architecture

```
frontend/           # Vue 3 + Vite (build → backend/web/dist)
backend/
  main.go           # serveur local + ouverture navigateur + jeton de session
  web/embed.go      # //go:embed dist  (frontend embarqué)
  internal/
    server/         # routeur HTTP + auth (jeton) + statique
    vault/          # coffre AES-256-GCM + Argon2id
    api/            # handlers REST (/api/*)
    sim/            # simulateur d'équipement (SSH réel + SNMP démo)
    history/        # snapshots de site + rapport
    configstore/    # configs par équipement + baseline/drift
    network/
      netinfo arp radar oui portfinder configdiff lldp macspoof diag neighbors
    models/         # DTO JSON partagés
```

## Tests

```bash
cd backend && go test ./...
```
