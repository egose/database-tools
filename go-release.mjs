const version = process.env.VERSION ?? 'localdev';
const sourceDateEpoch = Number(process.env.SOURCE_DATE_EPOCH ?? 0);

const binaries = [
  { name: 'mongo-archive', package: 'github.com/egose/database-tools/mongoarchive/main' },
  { name: 'mongo-unarchive', package: 'github.com/egose/database-tools/mongounarchive/main' },
  { name: 'postgres-archive', package: 'github.com/egose/database-tools/postgresarchive/main' },
  { name: 'postgres-unarchive', package: 'github.com/egose/database-tools/postgresunarchive/main' },
];

export default {
  toolName: 'database-tools',
  version,
  outputDir: 'dist',
  binaries: binaries.map((binary) => ({
    ...binary,
    linkerValues: [{ symbol: 'main.version', value: '{version} {os}-{arch}' }],
    versionCommand: {
      args: ['--version'],
      expectedOutput: `${binary.name} version: {version} {os}-{arch}\n`,
      match: 'exact',
    },
  })),
  targets: [
    { os: 'linux', arch: 'amd64' },
    { os: 'linux', arch: 'arm64' },
    { os: 'linux', arch: '386' },
    { os: 'linux', arch: 'arm' },
    { os: 'linux', arch: 'mips' },
    { os: 'linux', arch: 'mips64' },
    { os: 'windows', arch: 'amd64' },
    { os: 'windows', arch: '386' },
    { os: 'darwin', arch: 'amd64' },
    { os: 'darwin', arch: 'arm64' },
    { os: 'freebsd', arch: 'amd64' },
    { os: 'freebsd', arch: 'arm64' },
    { os: 'openbsd', arch: 'amd64' },
    { os: 'openbsd', arch: 'arm64' },
    { os: 'netbsd', arch: 'amd64' },
  ],
  buildFlags: ['-trimpath', '-buildvcs=false'],
  linkerFlags: ['-buildid='],
  archiveName: '{tool}-{os}-{arch}.tar.gz',
  checksumFile: `database-tools-${version}-sha256.txt`,
  additionalFiles: [{ source: 'LICENSE', destination: 'LICENSE' }],
  sourceDateEpoch,
  processLimits: { concurrency: 2 },
};
