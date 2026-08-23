import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Database Tools',
  tagline: 'MongoDB archive and restore tools for scheduled backups, cloud storage, and operational notifications.',
  favicon: 'img/favicon.svg',

  future: {
    v4: true,
  },

  url: 'https://database-tools.pages.dev/',
  baseUrl: '/',
  trailingSlash: true,
  organizationName: 'egose',
  projectName: 'database-tools',

  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/social-card.svg',
    navbar: {
      title: 'Database Tools',
      logo: {
        alt: 'Database Tools logo',
        src: 'img/logo.svg',
        srcDark: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'about',
          position: 'left',
          label: 'About',
        },
        {
          type: 'docSidebar',
          sidebarId: 'operations',
          position: 'left',
          label: 'Operations',
        },
        {
          type: 'docSidebar',
          sidebarId: 'deployment',
          position: 'left',
          label: 'Deployment',
        },
        {
          type: 'docSidebar',
          sidebarId: 'reference',
          position: 'left',
          label: 'Reference',
        },
        {
          href: 'https://github.com/egose/database-tools',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [],
      copyright: `Copyright © ${new Date().getFullYear()} Egose. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
    colorMode: {
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
