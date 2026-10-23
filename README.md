# PatchPulse

<div align="center">

**Open-Source Agentic Wiki & Repository Auto-Updater**

Automatically keep your gaming wikis, documentation, and data repositories updated with the latest patch notes.

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Node Version](https://img.shields.io/badge/node-%3E%3D20.0.0-brightgreen)](https://nodejs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-blue)](https://www.typescriptlang.org/)

</div>

## 🎯 Overview

PatchPulse monitors game changelog/patch note URLs, detects when updates go live, uses an LLM agent with tool-calling to interpret the changes, and automatically pushes updates (via GitHub Pull Requests or Wiki APIs) along with Discord notifications.

### Key Features

- **🤖 Agentic AI Integration**: LLM-powered agent that intelligently interprets patch notes and decides what to update
- **🔌 Modular Adapter Architecture**: Easy to extend with new integrations (GitHub, MediaWiki, Discord included)
- **📊 Web Dashboard**: Clean dark-mode UI for managing watched targets and viewing detection history
- **🔄 Automated Scheduling**: Cron-based periodic checks with configurable intervals
- **🐳 Docker Ready**: Multi-stage Docker build with persistent data volume
- **📝 Multiple Formats**: Supports HTML scraping, RSS/Atom feeds, and Markdown conversion
- **🔍 Hash-Based Diff Detection**: Efficient change detection using SHA-256 content hashing

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         PatchPulse                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│  │  Web Dashboard│    │   REST API    │    │   Scheduler  │ │
│  │  (Express +  │◄──►│   (Express)   │◄──►│  (node-cron) │ │
│  │   Vanilla JS)│    │               │    │              │ │
│  └──────────────┘    └──────────────┘    └──────┬───────┘ │
│                                                │            │
│                                                ▼            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Watcher / Scraper Engine                │  │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────────────┐  │  │
│  │  │ Fetcher  │ │  Parser  │ │  Hash Computer     │  │  │
│  │  └──────────┘ └──────────┘ └────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘ │
│                                                │            │
│                                                ▼            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   LLM Agent Layer                     │  │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────────────┐  │  │
│  │  │ Gemini   │ │  OpenAI  │ │  Tool Calling      │  │  │
│  │  │ Provider │ │ Provider │ │  (read, search,    │  │  │
│  │  └──────────┘ └──────────┘ │   patch, create)   │  │  │
│  └──────────────────────────────────────────────────────┘ │
│                                                │            │
│                                                ▼            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Adapter Architecture                   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌────────────┐  │  │
│  │  │   GitHub PR  │ │  MediaWiki   │ │  Discord   │  │  │
│  │  │   Adapter    │ │   Adapter    │ │  Adapter   │  │  │
│  │  └──────────────┘ └──────────────┘ └────────────┘  │  │
│  └──────────────────────────────────────────────────────┘ │
│                                                │            │
│                                                ▼            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            SQLite Database (WAL Mode)                │  │
│  │  ┌──────────┐ ┌──────────────┐ ┌──────────────┐  │  │
│  │  │ Targets  │ │ Patch History│ │ Agent Logs   │  │  │
│  │  └──────────┘ └──────────────┘ └──────────────┘  │  │
│  └──────────────────────────────────────────────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Node.js 20+ and npm
- Python 3.6+ (for building better-sqlite3 native module on non-Docker setups)
- Docker (optional, for containerized deployment - recommended to avoid native module build issues)

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/patchpulse.git
   cd patchpulse
   ```

2. **Install dependencies:**
   ```bash
   npm install
   ```

3. **Configure environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env with your API keys and configuration
   ```

4. **Build the project:**
   ```bash
   npm run build
   ```

5. **Start the server:**
   ```bash
   npm start
   ```

6. **Access the dashboard:**
   Open http://localhost:3000 in your browser

### Docker Deployment

```bash
# Build and start with Docker Compose
docker-compose up --build

# The application will be available at http://localhost:3000
# SQLite data persists in the Docker volume
```

## 📖 Usage

### Adding a Target

1. Navigate to the web dashboard
2. Click "Add Target"
3. Fill in the required fields:
   - **Name**: Descriptive name for the target
   - **URL**: Changelog/patch notes URL to monitor
   - **Adapter Type**: Choose from GitHub PR, MediaWiki, or Discord Webhook
   - **Adapter Config**: JSON configuration for the adapter (see examples below)
   - **Check Interval**: How often to check for updates (in seconds)

### Adapter Configuration Examples

#### GitHub PR Adapter
```json
{
  "owner": "your-username",
  "repo": "your-repo",
  "branch": "main",
  "token": "ghp_xxxxxxxxxxxx"
}
```

#### MediaWiki Adapter
```json
{
  "apiUrl": "https://wiki.example.com/api.php",
  "username": "your-username",
  "password": "your-password"
}
```

#### Discord Webhook Adapter
```json
{
  "webhookUrl": "https://discord.com/api/webhooks/xxxxxxxx/xxxxxxxx"
}
```

### Manual Scan

Click "Scan Now" on any target to trigger an immediate check for updates.

### Viewing History

- **Patches Tab**: View all detected patch notes with full content
- **Agent Logs Tab**: View agent actions, tool calls, and results

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `3000` | No |
| `HOST` | Server host | `0.0.0.0` | No |
| `DB_DIR` | Database directory | `./data` | No |
| `LLM_PROVIDER` | LLM provider (gemini/openai/openai-compatible) | `openai` | Yes |
| `LLM_API_KEY` | API key for LLM provider | - | Yes |
| `LLM_MODEL` | Model name | `gpt-4` | No |
| `LLM_BASE_URL` | Base URL for OpenAI-compatible endpoints | - | No |
| `GITHUB_TOKEN` | GitHub personal access token | - | For GitHub adapter |
| `GITHUB_DEFAULT_BRANCH` | Default branch name | `main` | No |
| `MEDIAWIKI_API_URL` | MediaWiki API endpoint | - | For MediaWiki adapter |
| `MEDIAWIKI_USERNAME` | MediaWiki username | - | For MediaWiki adapter |
| `MEDIAWIKI_PASSWORD` | MediaWiki password | - | For MediaWiki adapter |
| `MEDIAWIKI_API_TOKEN` | MediaWiki API token | - | For MediaWiki adapter |
| `DISCORD_WEBHOOK_URL` | Discord webhook URL | - | For Discord adapter |
| `USER_AGENT` | HTTP user agent | `PatchPulse/1.0` | No |
| `REQUEST_TIMEOUT` | Request timeout (ms) | `30000` | No |
| `MAX_RETRIES` | Max retry attempts | `3` | No |

## 🧩 Building Custom Adapters

PatchPulse's modular adapter architecture makes it easy to add new integrations. See [CONTRIBUTING.md](CONTRIBUTING.md) for a detailed guide on building custom adapters.

### Adapter Interface

```typescript
export abstract class BaseAdapter {
  abstract initialize(): Promise<void>;
  abstract readFile(path: string): Promise<string>;
  abstract searchFiles(query: string, filePattern?: string): Promise<string[]>;
  abstract updateFile(path: string, content: string, commitMessage: string): Promise<void>;
  abstract commitChanges(message: string): Promise<string>;
  abstract getDiffUrl(): string;
  abstract getType(): string;
}
```

## 📁 Project Structure

```
patchpulse/
├── src/
│   ├── index.ts                # Entry point
│   ├── config/                 # Environment configuration
│   │   └── env.ts
│   ├── db/                     # SQLite database layer
│   │   ├── schema.ts
│   │   ├── connection.ts
│   │   └── queries.ts
│   ├── watcher/                # Scraper and scheduler
│   │   ├── fetcher.ts
│   │   ├── html-parser.ts
│   │   ├── markdown-converter.ts
│   │   ├── rss-parser.ts
│   │   ├── hash-computer.ts
│   │   ├── scheduler.ts
│   │   └── diff-detector.ts
│   ├── agent/                  # LLM agent layer
│   │   ├── agent.ts
│   │   ├── tools.ts
│   │   ├── prompts.ts
│   │   ├── parser.ts
│   │   ├── gemini-provider.ts
│   │   └── openai-provider.ts
│   ├── adapters/               # Adapter implementations
│   │   ├── BaseAdapter.ts
│   │   ├── GitHubPRAdapter.ts
│   │   ├── MediaWikiAdapter.ts
│   │   ├── DiscordWebhookAdapter.ts
│   │   └── index.ts
│   └── api/                    # Express API routes
│       ├── index.ts
│       ├── targets.ts
│       ├── triggers.ts
│       └── logs.ts
├── public/
│   └── index.html              # Web dashboard
├── examples/
│   └── mock-game-data/         # Example data for testing
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── package.json
├── tsconfig.json
└── README.md
```

## 🔧 Troubleshooting

### better-sqlite3 Build Errors

If you encounter build errors when installing dependencies (especially on Windows), `better-sqlite3` requires Python to compile native modules. You have two options:

**Option 1: Use Docker (Recommended)**
```bash
docker-compose up --build
```
This avoids native module compilation entirely.

**Option 2: Install Python and Build Tools**

On Windows:
1. Install Python 3.6+ from https://www.python.org/downloads/
2. Install Visual Studio Build Tools: https://visualstudio.microsoft.com/visual-cpp-build-tools/
3. Then run:
   ```bash
   npm install
   ```

On Linux/macOS:
```bash
# Install Python and build tools
sudo apt-get install python3 build-essential  # Ubuntu/Debian
brew install python3  # macOS

npm install
```

### Database Lock Issues

If you encounter database lock errors, ensure only one instance of PatchPulse is running, or use Docker Compose which handles this automatically.

## 🧪 Testing

### Local Testing with Mock Data

The `examples/mock-game-data/` directory contains sample files for local testing:

```bash
# Serve the mock HTML locally (e.g., using Python)
cd examples/mock-game-data
python -m http.server 8080

# Add a target in the dashboard pointing to http://localhost:8080/patch-notes.html
```

### Running Tests

```bash
npm test
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:

- Building custom adapters
- Code style and conventions
- Submitting pull requests
- Reporting issues

## 📄 License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [better-sqlite3](https://github.com/WiseLibs/better-sqlite3) for fast SQLite operations
- HTML parsing powered by [cheerio](https://cheerio.js.org/)
- Markdown conversion via [turndown](https://github.com/mixmark-io/turndown)
- LLM integration with [OpenAI](https://github.com/openai/openai-node) and [Google Generative AI](https://github.com/google/generative-ai-js)
- GitHub API via [Octokit](https://github.com/octokit/rest.js/)

## 📞 Support

- 📖 [Documentation](https://github.com/yourusername/patchpulse/wiki)
- 🐛 [Issue Tracker](https://github.com/yourusername/patchpulse/issues)
- 💬 [Discussions](https://github.com/yourusername/patchpulse/discussions)

---

Made with ❤️ by the PatchPulse community
