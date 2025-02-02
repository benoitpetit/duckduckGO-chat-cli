# 🦆 DuckDuckGo AI Chat CLI

**Un outil CLI puissant pour interagir avec l'IA de DuckDuckGo**  
_Intégration de contexte avancée, multi-modèles et productivité augmentée_

## ✨ Fonctionnalités phares

| **Chat intelligent**    | **Gestion de contexte**   | **Intégrations**   |
| ----------------------- | ------------------------- | ------------------ |
| ▶️ Streaming temps réel | 🔍 Recherche web intégrée | 📂 Fichiers locaux |
| 🤖 4 modèles IA         | 🌐 Extraction web         | 📦 Export Markdown |
| 🔄 Régénération         | 🧹 Nettoyage intelligent  | 🕸️ Rendering JS    |
| 🎨 Sortie colorée       | ⏳ Historique             | 🔐 Gestion tokens  |

## 🧠 Modèles supportés

### `GPT-4o mini` (_Recommandé_)

- **Optimisé pour** : Réponses rapides, généralistes
- **Cas d'usage** : Discussions courantes, brainstorming
- **Limite contexte** : 4K tokens

### `Claude 3 Haiku`

- **Spécialité** : Analyse de données structurées
- **Force** : Compréhension contextuelle profonde
- **Bonus** : Supporte les prompts complexes

### `Llama 3.1 70B`

- **Pour qui** : Développeurs/Data Scientists
- **Atout** : Génération de code/analyse technique
- **Configuration** : 8GB RAM minimum

### `Mixtral 8x7B`

- **Expertise** : Sujets spécialisés (médecine, droit)
- **Avantage** : Synthèse multi-sources
- **Performance** : Latence légèrement plus élevée

## 🛠️ Installation

### Prérequis

- Go 1.21+ (`go version`)
- Chrome/Chromium 115+ (`chromium-browser --version`)
- 500MB d'espace disque

### Méthodes d'installation

```bash
# Linux
curl -LO https://github.com/benoitpetit/duckduckGO-chat-cli/releases/latest/download/duckduckgo-chat-cli_linux_amd64
chmod +x duckduckgo-chat-cli_linux_amd64

# macOS
brew tap benoitpetit/cli && brew install duckduckgo-chat-cli
```

**2. Compilation depuis les sources :**

```bash
git clone https://github.com/benoitpetit/duckduckGO-chat-cli
cd duckduckGO-chat-cli
go build -ldflags "-s -w" -o ddg-chat
```

## 🚀 Utilisation avancée

### Workflow typique

```bash
./ddg-chat
> Accept terms? [yes/no] yes
> Choisir modèle (1-4): 2

[Claude 3 Haiku activé]
/user : /search meilleures pratiques Rust 2025
[+] 10 résultats ajoutés
/user : /file ~/project/src/lib.rs
[+] Fichier analysé (1.2KB)
/user : Comment améliorer cette implémentation ?
AI : █ Génération en cours...
```

### Commandes essentielles

| Commande          | Exemple                          | Résultat              |
| ----------------- | -------------------------------- | --------------------- |
| `/search <query>` | `/search GPT-5 spéculations`     | Injecte 10 résultats  |
| `/file <chemin>`  | `/file /tmp/notes.md`            | Ajoute le contenu     |
| `/url <lien>`     | `/url https://arxiv.org/abs/123` | Extrait le contenu    |
| `/clear`          | `/clear`                         | Réinitialise contexte |
| `/markdown`       | `/markdown`                      | Génère export MD      |
| `/extract`        | `/extract`                       | Crée synthèse         |

## 🔧 Configuration avancée

### Variables d'environnement

```bash
export DDG_TIMEOUT=60        # Timeout des requêtes (secondes)
export CHROMEDP_PATH=/usr/bin/chromium  # Chemin personnalisé Chrome
export MAX_CONTEXT=5000      # Limite de tokens contextuels
```

### Format d'export Markdown

````markdown
# Conversation du 15/03/2024

## Contexte recherche (15/03 14:30)

```rust
▸ Rust Security Audit Guide
  "Best practices for unsafe code..."
  https://rustsec.org
```

## Message utilisateur (15/03 14:32)

Comment sécuriser ce bloc unsafe ?

## Réponse AI (15/03 14:33)

1. Utiliser `SafeWrapper` pour les pointeurs bruts...

````

## 🚨 Dépannage

**Problème** : Échec d'extraction web
**Solution** :
```bash
# Vérifier la version de Chrome
chromium-browser --version  # Doit afficher ≥ 115.0.5790.110

# Lancer en mode debug
DDG_DEBUG=1 ./ddg-chat
````

**Problème** : Token VQD expiré  
**Solution** :

```bash
/user : /clear  # Régénère automatiquement le token
```

**Problème** : Latence élevée  
**Solution** :

- Changer de modèle (`/clear` puis choisir GPT-4o mini)
- Réduire la taille du contexte (`export MAX_CONTEXT=3000`)

## 📜 Licence & Éthique

- **Licence** : MIT License
- **Collecte de données** : Aucune donnée personnelle stockée
- **Attention** : Les sorties IA peuvent contenir des erreurs - toujours vérifier les faits critiques

_Ce projet n'est pas affilié à DuckDuckGo - utilisez à vos risques_

> Made with ♥ par Benoit Petit - [Contribution guide](CONTRIBUTING.md)
