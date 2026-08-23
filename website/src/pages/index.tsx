import { useEffect, useState, type ReactNode } from 'react';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';

const features = [
  {
    title: 'Archive to cloud storage',
    description:
      'Run MongoDB dumps locally, then upload managed archives to Azure Blob Storage, AWS S3, Google Cloud Storage, or a local path.',
  },
  {
    title: 'Restore with explicit safety checks',
    description:
      'Download the selected backup, extract only safe archive entries, run mongorestore, and stop before post-restore updates if restore errors occur.',
  },
  {
    title: 'Operate scheduled backups',
    description:
      'Use one-shot or cron mode, retention, failure-only notifications, and multi-backend upload contracts designed for production operations.',
  },
];

const paths = [
  {
    title: 'Quick Start',
    label: 'Start here',
    description: 'Install the binaries, verify them, and run the first archive and restore commands.',
    to: '/docs/about/quick-start/',
  },
  {
    title: 'Archive Operations',
    label: 'Backup flow',
    description: 'Understand backup prefixes, generated archive names, retention, and multi-backend behavior.',
    to: '/docs/operations/archive/',
  },
  {
    title: 'CLI Flag Reference',
    label: 'Reference',
    description: 'Find the tested flag and environment variable reference for both command-line tools.',
    to: '/docs/reference/cli-flags/',
  },
];

const codeSample = `mongo-archive \\
  --uri="mongodb://user:pass@cluster0.mongodb.net/" # pragma: allowlist secret \\
  --db=app \\
  --aws-region=us-east-1 \\
  --aws-bucket=database-backups \\
  --backup-prefix=mongo-archive/ \\
  --slack-notify-on-failure-only

mongo-unarchive \\
  --uri="mongodb://localhost:27017" \\
  --db=app \\
  --aws-region=us-east-1 \\
  --aws-bucket=database-backups`;

function Hero(): ReactNode {
  const { siteConfig } = useDocusaurusContext();

  return (
    <section className="heroBanner">
      <div className="heroInner">
        <div className="heroPanel">
          <span className="eyebrow">MongoDB backup and restore CLIs</span>
          <h1 className="heroTitle">Back up MongoDB with contracts operators can trust.</h1>
          <p className="heroLead">
            {siteConfig.title} provides `mongo-archive` and `mongo-unarchive`: focused command-line tools for dumping,
            uploading, downloading, restoring, retaining, and notifying around MongoDB backups.
          </p>
          <div className="heroActions">
            <Link className="buttonPrimary" to="/docs/about/quick-start/">
              Open Quick Start
            </Link>
            <Link className="buttonSecondary" to="/docs/operations/storage-backends/">
              Browse Storage Backends
            </Link>
          </div>
        </div>

        <div className="terminalCard" aria-label="Example archive and restore commands">
          <div className="terminalChrome">
            <div className="terminalDots" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
            <span>backup-restore.sh</span>
          </div>
          <pre>
            <code>{codeSample}</code>
          </pre>
        </div>
      </div>
    </section>
  );
}

function Features(): ReactNode {
  return (
    <section className="sectionBlock">
      <div className="sectionInner">
        <div className="sectionHeader">
          <span className="eyebrow">Core workflows</span>
          <h2>Two tools, one operational backup loop.</h2>
          <p>
            The docs are organized around the same boundaries as the codebase: archive, restore, storage, notifications,
            deployment, and release policy.
          </p>
        </div>
        <div className="cardGrid">
          {features.map((feature, index) => (
            <article className="featureCard" key={feature.title}>
              <div className="cardNumber">0{index + 1}</div>
              <h3>{feature.title}</h3>
              <p>{feature.description}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function Operations(): ReactNode {
  return (
    <section className="operationsBand sectionBlock">
      <div className="sectionInner">
        <div className="sectionHeader">
          <span className="eyebrow">Built for release discipline</span>
          <h2>Documented behavior mirrors tested contracts.</h2>
        </div>
        <div className="statsGrid">
          <div className="statCard">
            <strong>2</strong>
            <span>CLI tools</span>
          </div>
          <div className="statCard">
            <strong>4</strong>
            <span>storage targets</span>
          </div>
          <div className="statCard">
            <strong>4</strong>
            <span>notification backends</span>
          </div>
          <div className="statCard">
            <strong>1</strong>
            <span>generated flag reference</span>
          </div>
        </div>
      </div>
    </section>
  );
}

function Pathways(): ReactNode {
  return (
    <section className="sectionBlock">
      <div className="sectionInner">
        <div className="cardGrid">
          {paths.map((path) => (
            <Link className="pathCard" key={path.title} to={path.to}>
              <div className="cardNumber">{path.label}</div>
              <h3>{path.title}</h3>
              <p>{path.description}</p>
            </Link>
          ))}
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  const { siteConfig } = useDocusaurusContext();
  const [, setThemeTick] = useState(0);

  useEffect(() => {
    const observer = new MutationObserver(() => setThemeTick((tick) => tick + 1));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });

    return () => observer.disconnect();
  }, []);

  return (
    <Layout
      title={siteConfig.title}
      description="Documentation for mongo-archive and mongo-unarchive, the MongoDB backup and restore tools in database-tools."
    >
      <main>
        <Hero />
        <Features />
        <Operations />
        <Pathways />
      </main>
    </Layout>
  );
}
