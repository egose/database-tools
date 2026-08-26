export default {
  cwd: '.',
  compose: {
    files: ['sandbox/docker-compose.yml', 'sandbox/docker-compose-ci.yml'],
    envFile: '.env.example',
    projectName: 'database-tools',
  },
  // No host bind-mounts – sandbox uses named Docker volumes
  // (postgres_data, mongodb_data, etc.) to avoid EACCES on
  // sandbox/mnt/* (postgres/mongo create files as root). Volumes are
  // removed via `cleanup.volumes: true` + `docker compose down -v`.
  readiness: [
    { type: 'tcp', host: '127.0.0.1', port: 5432 },
    { type: 'tcp', host: '127.0.0.1', port: 27017 },
    { type: 'tcp', host: '127.0.0.1', port: 9000 },
    { type: 'http', url: 'http://127.0.0.1:9000/minio/health/live' },
    { type: 'tcp', host: '127.0.0.1', port: 9001 },
    { type: 'tcp', host: '127.0.0.1', port: 10000 },
    { type: 'tcp', host: '127.0.0.1', port: 8080 },
    { type: 'service-completed', service: 'minio-init' },
    {
      type: 'command',
      executable: 'mongosh',
      args: [
        'mongodb://mongodb:mongodb@127.0.0.1:27017/?authSource=admin', // pragma: allowlist secret
        '--quiet',
        '--eval',
        'db.adminCommand({ ping: 1 })',
      ],
      timeoutMs: 30000,
    },
    {
      type: 'command',
      executable: 'psql',
      args: [
        '--no-psqlrc',
        '-X',
        '--quiet',
        '--tuples-only',
        '--no-align',
        '--host=127.0.0.1',
        '--port=5432',
        '--username=postgres',
        '--dbname=integration',
        '--command=SELECT 1',
      ],
      env: { PGPASSWORD: 'postgres' }, // pragma: allowlist secret
      timeoutMs: 30000,
    },
  ],
  test: {
    executable: 'bats',
    args: ['--print-output-on-failure', 'test/test.bats'],
  },
  evidence: {
    directory: 'sandbox/test-logs',
    capture: 'always',
  },
  cleanup: {
    volumes: true,
    removeOrphans: true,
  },
  timeouts: {
    startupMs: 120000,
    readinessMs: 120000,
    testMs: 600000,
    cleanupMs: 30000,
  },
};
