const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

// Load .env if it exists
let port = '3001';
const envPath = path.join(__dirname, '.env');
if (fs.existsSync(envPath)) {
  const envContent = fs.readFileSync(envPath, 'utf8');
  const match = envContent.match(/^PORT\s*=\s*(\d+)/m);
  if (match) {
    port = match[1];
  }
}

// Override with process.env.PORT if specified
if (process.env.PORT) {
  port = process.env.PORT;
}

const args = process.argv.slice(2);
const command = args[0] === 'start' ? 'start' : 'dev';

const npx = process.platform === 'win32' ? 'npx.cmd' : 'npx';
const child = spawn(npx, ['next', command, '-p', port], { stdio: 'inherit', shell: true });
child.on('exit', (code) => process.exit(code));
