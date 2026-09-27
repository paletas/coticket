#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { pipeline } = require('stream');
const { promisify } = require('util');

const streamPipeline = promisify(pipeline);

// Read version from VERSION file
const VERSION = fs.readFileSync(path.join(__dirname, '..', 'VERSION'), 'utf8').trim();

// Determine platform and architecture
const platform = process.platform; // darwin, linux, win32
const arch = process.arch; // x64, arm64

// Map Node.js platform/arch to binary naming convention
const platformMap = {
  'win32': 'windows',
  'darwin': 'darwin',
  'linux': 'linux'
};

const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

const goPlatform = platformMap[platform];
const goArch = archMap[arch];

if (!goPlatform || !goArch) {
  console.error(`Unsupported platform/architecture: ${platform}/${arch}`);
  process.exit(1);
}

// Construct binary name
const binaryName = platform === 'win32' 
  ? `coticket-mcp-${goPlatform}-${goArch}.exe`
  : `coticket-mcp-${goPlatform}-${goArch}`;

// Download URL
const REPO = 'paletas/coticket';
const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;

// Binary destination
const binDir = path.join(__dirname, '..', 'bin');
const binaryPath = path.join(binDir, binaryName);

// Skip download if binary already exists
if (fs.existsSync(binaryPath)) {
  console.log(`Binary already exists: ${binaryPath}`);
  process.exit(0);
}

console.log(`Downloading CoTicket MCP v${VERSION} for ${goPlatform}/${goArch}...`);
console.log(`URL: ${downloadUrl}`);

// Ensure bin directory exists
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Download binary
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    // Follow redirects
    https.get(url, { headers: { 'User-Agent': 'npm-postinstall' } }, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        return downloadFile(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Download failed with status ${response.statusCode}: ${url}`));
        return;
      }

      const file = fs.createWriteStream(dest);
      streamPipeline(response, file)
        .then(() => {
          // Make binary executable on Unix-like systems
          if (platform !== 'win32') {
            fs.chmodSync(dest, 0o755);
          }
          resolve();
        })
        .catch(reject);
    }).on('error', reject);
  });
}

downloadFile(downloadUrl, binaryPath)
  .then(() => {
    console.log(`✓ Downloaded successfully: ${binaryPath}`);
    
    // Make sure it's executable
    if (platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
    }
  })
  .catch((error) => {
    console.error(`✗ Failed to download binary: ${error.message}`);
    console.error(`\nPlease ensure release v${VERSION} exists at:`);
    console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);
    console.error(`\nOr install manually by building from source:`);
    console.error(`  git clone https://github.com/${REPO}.git`);
    console.error(`  cd coticket`);
    console.error(`  go build -o ${binaryName} ./cmd/coticket-mcp`);
    process.exit(1);
  });
