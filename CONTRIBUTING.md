# Contributing to PatchPulse

Thank you for your interest in contributing to PatchPulse! This guide will help you get started with building custom adapters and contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Building Custom Adapters](#building-custom-adapters)
- [Code Style](#code-style)
- [Submitting Changes](#submitting-changes)
- [Testing](#testing)

## Getting Started

### Prerequisites

- Node.js 20+
- TypeScript 5.6+
- Python 3.6+ (for building better-sqlite3 native module)
- Git

**Note**: If you encounter build errors with `better-sqlite3`, see the Troubleshooting section in README.md or use Docker for development.

### Development Setup

1. **Fork and clone the repository:**
   ```bash
   git clone https://github.com/yourusername/patchpulse.git
   cd patchpulse
   ```

2. **Install dependencies:**
   ```bash
   npm install
   ```

3. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make your changes and build:**
   ```bash
   npm run build
   npm start
   ```

## Building Custom Adapters

One of the core design goals of PatchPulse is to make it easy for the community to add new integrations through a modular adapter architecture.

### Understanding the Adapter Interface

All adapters must extend the `BaseAdapter` abstract class and implement the following methods:

```typescript
import { BaseAdapter, AdapterConfig } from './BaseAdapter.js';

export class YourCustomAdapter extends BaseAdapter {
  constructor(config: AdapterConfig) {
    super(config);
    // Initialize your adapter-specific properties
  }

  async initialize(): Promise<void> {
    // Set up connections, authenticate, etc.
  }

  async readFile(path: string): Promise<string> {
    // Read and return file content from your target system
  }

  async searchFiles(query: string, filePattern?: string): Promise<string[]> {
    // Search for files matching the query and optional pattern
    // Return array of file paths
  }

  async updateFile(path: string, content: string, commitMessage: string): Promise<void> {
    // Stage a file update (don't commit yet)
  }

  async commitChanges(message: string): Promise<string> {
    // Commit all staged changes and return a URL to view the diff
  }

  getDiffUrl(): string {
    // Return a URL where humans can review the changes
  }

  getType(): string {
    // Return a unique identifier for your adapter type
  }
}
```

### Step-by-Step Adapter Example

Let's build a simple adapter for a hypothetical "GitLab" integration:

#### 1. Create the adapter file

Create `src/adapters/GitLabAdapter.ts`:

```typescript
import { BaseAdapter, AdapterConfig } from './BaseAdapter.js';

interface GitLabConfig extends AdapterConfig {
  projectId: string;
  token: string;
  branch?: string;
  apiUrl?: string;
}

export class GitLabAdapter extends BaseAdapter {
  private projectId: string;
  private token: string;
  private branch: string;
  private apiUrl: string;
  private stagedFiles: Map<string, string> = new Map();

  constructor(config: GitLabConfig) {
    super(config);
    this.projectId = config.projectId;
    this.token = config.token;
    this.branch = config.branch || 'main';
    this.apiUrl = config.apiUrl || 'https://gitlab.com/api/v4';
  }

  async initialize(): Promise<void> {
    // Verify the token works by fetching project info
    const response = await fetch(
      `${this.apiUrl}/projects/${encodeURIComponent(this.projectId)}`,
      {
        headers: { 'PRIVATE-TOKEN': this.token },
      }
    );

    if (!response.ok) {
      throw new Error(`GitLab authentication failed: ${response.statusText}`);
    }
  }

  async readFile(filePath: string): Promise<string> {
    const encodedPath = encodeURIComponent(filePath);
    const response = await fetch(
      `${this.apiUrl}/projects/${encodeURIComponent(this.projectId)}/repository/files/${encodedPath}/raw?ref=${this.branch}`,
      {
        headers: { 'PRIVATE-TOKEN': this.token },
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to read file: ${response.statusText}`);
    }

    return response.text();
  }

  async searchFiles(query: string, filePattern?: string): Promise<string[]> {
    const response = await fetch(
      `${this.apiUrl}/projects/${encodeURIComponent(this.projectId)}/search?scope=blobs&search=${encodeURIComponent(query)}`,
      {
        headers: { 'PRIVATE-TOKEN': this.token },
      }
    );

    if (!response.ok) {
      throw new Error(`Search failed: ${response.statusText}`);
    }

    const data = await response.json();
    return data.map((result: any) => result.filename);
  }

  async updateFile(filePath: string, content: string, commitMessage: string): Promise<void> {
    // Stage the file for later commit
    this.stagedFiles.set(filePath, content);
  }

  async commitChanges(message: string): Promise<string> {
    if (this.stagedFiles.size === 0) {
      throw new Error('No files staged for commit');
    }

    // Create a new branch
    const branchName = `patchpulse-${Date.now()}`;
    await fetch(
      `${this.apiUrl}/projects/${encodeURIComponent(this.projectId)}/repository/branches`,
      {
        method: 'POST',
        headers: {
          'PRIVATE-TOKEN': this.token,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          branch: branchName,
          ref: this.branch,
        }),
      }
    );

    // Commit each file
    for (const [filePath, content] of this.stagedFiles.entries()) {
      const encodedPath = encodeURIComponent(filePath);
      await fetch(
        `${this.apiUrl}/projects/${encodeURIComponent(this.projectId)}/repository/files/${encodedPath}`,
        {
          method: 'PUT',
          headers: {
            'PRIVATE-TOKEN': this.token,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            branch: branchName,
            content: Buffer.from(content).toString('base64'),
            commit_message: message,
            encoding: 'base64',
          }),
        }
      );
    }

    // Create a merge request
    const mrResponse = await fetch(
      `${this.apiUrl}/projects/${encodeURIComponent(this.projectId)}/merge_requests`,
      {
        method: 'POST',
        headers: {
          'PRIVATE-TOKEN': this.token,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          source_branch: branchName,
          target_branch: this.branch,
          title: message,
          description: `Automated update by PatchPulse\n\n${message}`,
        }),
      }
    );

    const mrData = await mrResponse.json();
    this.stagedFiles.clear();

    return mrData.web_url;
  }

  getDiffUrl(): string {
    if (this.stagedFiles.size === 0) {
      return '';
    }
    return `${this.apiUrl.replace('/api/v4', '')}/${this.projectId}/-/merge_requests`;
  }

  getType(): string {
    return 'gitlab-mr';
  }
}
```

#### 2. Register the adapter

Update `src/adapters/index.ts`:

```typescript
import { BaseAdapter } from './BaseAdapter.js';
import { GitHubPRAdapter } from './GitHubPRAdapter.js';
import { MediaWikiAdapter } from './MediaWikiAdapter.js';
import { DiscordWebhookAdapter } from './DiscordWebhookAdapter.js';
import { GitLabAdapter } from './GitLabAdapter.js'; // Import your adapter

export type { AdapterConfig } from './BaseAdapter.js';

export function createAdapter(type: string, config: any): BaseAdapter {
  switch (type) {
    case 'github-pr':
      return new GitHubPRAdapter(config);
    case 'mediawiki':
      return new MediaWikiAdapter(config);
    case 'discord-webhook':
      return new DiscordWebhookAdapter(config);
    case 'gitlab-mr': // Add your adapter case
      return new GitLabAdapter(config);
    default:
      throw new Error(`Unknown adapter type: ${type}`);
  }
}

export { BaseAdapter, GitHubPRAdapter, MediaWikiAdapter, DiscordWebhookAdapter, GitLabAdapter };
```

#### 3. Update the web dashboard (optional)

If you want your adapter to appear in the dropdown in `public/index.html`:

```html
<select id="target-adapter" required class="w-full p-2 rounded bg-slate-800 border border-slate-600 text-white">
  <option value="github-pr">GitHub PR</option>
  <option value="mediawiki">MediaWiki</option>
  <option value="discord-webhook">Discord Webhook</option>
  <option value="gitlab-mr">GitLab MR</option> <!-- Add your option -->
</select>
```

#### 4. Test your adapter

Create a test target in the dashboard with your new adapter type and verify it works correctly.

### Adapter Best Practices

1. **Error Handling**: Always provide clear error messages that help users debug configuration issues
2. **Idempotency**: Ensure operations can be safely retried
3. **Staging Pattern**: Use the staging pattern (stage files, then commit) to allow for batched changes
4. **Authentication**: Validate credentials during `initialize()` rather than waiting until the first operation
5. **Rate Limiting**: Be aware of API rate limits and implement backoff if necessary
6. **Type Safety**: Define TypeScript interfaces for your adapter's configuration

## Code Style

PatchPulse follows these conventions:

- **TypeScript**: Strict mode enabled, no implicit any
- **Formatting**: Use 2-space indentation
- **Naming**: camelCase for variables/functions, PascalCase for classes/types
- **Comments**: Minimal inline comments; use descriptive names instead
- **Imports**: Use ES module syntax (`.js` extensions for TypeScript)

### Example

```typescript
// Good
async function fetchUserData(userId: string): Promise<User> {
  const response = await fetch(`/api/users/${userId}`);
  return response.json();
}

// Avoid
async function get(id) {
  const r = await fetch('/api/users/' + id);
  return r.json();
}
```

## Submitting Changes

1. **Update documentation**: If your change affects user-facing behavior, update README.md
2. **Add tests**: Write tests for new functionality
3. **Build successfully**: Ensure `npm run build` completes without errors
4. **Commit message**: Use clear, descriptive commit messages
5. **Pull request**: Create a PR with a description of your changes

### Commit Message Format

```
feat: add GitLab adapter implementation
fix: resolve memory leak in scheduler
docs: update adapter configuration examples
```

## Testing

### Running Tests

```bash
npm test
```

### Manual Testing Checklist

- [ ] Application builds without errors
- [ ] Web dashboard loads and functions correctly
- [ ] Can add/edit/delete targets
- [ ] Manual scan triggers work
- [ ] Patch detection logs are visible
- [ ] Adapter-specific functionality works (e.g., PR creation, wiki edits)

### Testing Adapters

For adapter-specific testing:

1. Create a test repository/wiki
2. Configure a target with your adapter
3. Trigger a manual scan
4. Verify the expected changes were made
5. Check the diff URL to review changes

## Questions?

If you have questions about contributing or building adapters:

- Open an issue on GitHub
- Start a discussion in the Discussions tab
- Check existing adapters for reference implementations

Thank you for contributing to PatchPulse! 🎉
