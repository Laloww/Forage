#!/usr/bin/env node

import { createWriteStream, chmodSync, existsSync, mkdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { get } from "https";
import { execSync } from "child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const VERSION = "0.2.0";
const REPO = "Laloww/Forage";

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  const platforms = {
    "darwin-arm64": "darwin-arm64",
    "darwin-x64": "darwin-amd64",
    "linux-arm64": "linux-arm64",
    "linux-x64": "linux-amd64",
    "win32-x64": "windows-amd64",
  };

  const key = `${platform}-${arch}`;
  const mapped = platforms[key];
  if (!mapped) {
    console.error(`Unsupported platform: ${key}`);
    console.error("Supported: darwin-arm64, darwin-x64, linux-arm64, linux-x64, win32-x64");
    console.error("\nYou can build from source: go install github.com/Laloww/Forage/cmd/forage@latest");
    process.exit(1);
  }
  return mapped;
}

function getBinaryName() {
  return process.platform === "win32" ? "forage.exe" : "forage-bin";
}

async function download(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (url) => {
      get(url, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          follow(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: HTTP ${res.statusCode} from ${url}`));
          return;
        }
        const file = createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => {
          file.close();
          resolve();
        });
        file.on("error", reject);
      }).on("error", reject);
    };
    follow(url);
  });
}

async function tryGoBuild() {
  try {
    execSync("go version", { stdio: "ignore" });
    console.log("Go found, building from source...");
    const binaryPath = join(__dirname, getBinaryName());
    execSync(`go build -o "${binaryPath}" github.com/Laloww/Forage/cmd/forage@latest`, {
      stdio: "inherit",
    });
    chmodSync(binaryPath, 0o755);
    return true;
  } catch {
    return false;
  }
}

async function main() {
  const platform = getPlatform();
  const binaryName = getBinaryName();
  const binaryPath = join(__dirname, binaryName);

  if (existsSync(binaryPath)) {
    console.log("forage binary already exists, skipping download.");
    return;
  }

  const ext = process.platform === "win32" ? ".exe" : "";
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/forage-${platform}${ext}`;

  console.log(`Downloading forage v${VERSION} for ${platform}...`);

  try {
    await download(url, binaryPath);
    chmodSync(binaryPath, 0o755);
    console.log("forage installed successfully!");
  } catch (err) {
    console.warn(`Download failed: ${err.message}`);
    console.log("Attempting to build from source with Go...");

    if (await tryGoBuild()) {
      console.log("forage built from source successfully!");
    } else {
      console.error("\nCould not install forage.");
      console.error("Options:");
      console.error("  1. Create a GitHub release: https://github.com/Laloww/Forage/releases");
      console.error("  2. Install Go and run: go install github.com/Laloww/Forage/cmd/forage@latest");
      process.exit(1);
    }
  }
}

main();
