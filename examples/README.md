# Example Mock Data

This directory contains sample data for testing PatchPulse locally without connecting to real services.

## Files

### `game-stats.json`
Sample game data file representing character stats and weapons. This simulates a JSON data file that might be in a game repository or wiki.

### `patch-notes.html`
Sample HTML patch notes page that simulates a game's changelog. This can be used to test the HTML parsing and Markdown conversion features.

## Testing Locally

To test PatchPulse with this mock data:

1. **Start the PatchPulse server:**
   ```bash
   npm run dev
   ```

2. **Access the web dashboard:**
   Open http://localhost:3000 in your browser

3. **Add a test target:**
   - Click "Add Target"
   - Name: "Mock Game Test"
   - URL: Use a local file path or serve the HTML via a local server
   - Adapter: GitHub PR (with a test repo) or MediaWiki (with a test wiki)
   - Adapter Config: Configure with your test repository/wiki credentials

4. **Manual scan:**
   - Click "Scan Now" on the target to trigger a manual check
   - View the detected patches in the "Patches" tab

## Notes

- For production use, you would replace these mock files with real URLs from game changelogs
- The adapter configuration will need to match your actual repository or wiki setup
- Ensure you have valid API tokens/credentials configured in your `.env` file
