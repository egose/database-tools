const version = process.env.VERSION ?? 'localdev';
const sourceDateEpoch = Number(process.env.SOURCE_DATE_EPOCH ?? 0);

export default {
  toolName: 'database-tools',
  version,
  outputDir: 'dist',
  binaries: [
    {
      name: 'mongo-archive',
      package: 'mongoarchive/main/mongoarchive.go',
      linkerValues: [{ symbol: 'main.version', value: '{version} {os}-{arch}' }],
      versionCommand: {
        args: ['--version'],
        expectedOutput: 'mongo-archive version: {version} {os}-{arch}\n',
        match: 'exact',
      },
    },
    {
      name: 'mongo-unarchive',
      package: 'mongounarchive/main/mongounarchive.go',
      linkerValues: [{ symbol: 'main.version', value: '{version} {os}-{arch}' }],
      versionCommand: {
        args: ['--version'],
        expectedOutput: 'mongo-unarchive version: {version} {os}-{arch}\n',
        match: 'exact',
      },
    },
  ],
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
