#!/usr/bin/env node

import { execFileSync } from "child_process";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { existsSync } from "fs";

const __dirname = dirname(fileURLToPath(import.meta.url));

function getBinaryPath() {
  const name = process.platform === "win32" ? "forage.exe" : "forage-bin";
  const local = join(__dirname, name);
  if (existsSync(local)) {
    return local;
  }

  // Fallback: try system PATH
  return "forage";
}

const binary = getBinaryPath();
const args = process.argv.slice(2);

// Default to MCP mode when called with no args (for npx/claude mcp add)
if (args.length === 0) {
  args.push("mcp");
}

try {
  execFileSync(binary, args, { stdio: "inherit" });
} catch (err) {
  if (err.status != null) {
    process.exit(err.status);
  }
  console.error(`Failed to run forage: ${err.message}`);
  console.error("Try: go install github.com/Laloww/Forage/cmd/forage@latest");
  process.exit(1);
}
