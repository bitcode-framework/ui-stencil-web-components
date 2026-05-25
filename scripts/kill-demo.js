const { execSync } = require('child_process');
const PORT = process.env.DEMO_PORT || 3333;

try {
  const result = execSync(
    `netstat -ano | findstr ":${PORT}" | findstr "LISTENING"`,
    { encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] }
  );
  const match = result.match(/LISTENING\s+(\d+)/);
  if (match) {
    const pid = match[1];
    console.log(`Killing process ${pid} on port ${PORT}...`);
    try {
      execSync(`powershell -Command "Stop-Process -Id ${pid} -Force"`, { stdio: 'pipe' });
      console.log(`Killed ${pid}.`);
    } catch {
      console.log(`Could not kill ${pid}. Try: taskkill /PID ${pid} /F`);
      process.exit(1);
    }
  } else {
    console.log(`Port ${PORT} is free.`);
  }
} catch {
  console.log(`Port ${PORT} is free.`);
}
