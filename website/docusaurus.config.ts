import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'MatrixHub',
  tagline: 'The Open Source Hub for AI Models',
  favicon: 'img/favicon.ico',

  // future: {
  //   v4: true,
  // },

  url: 'https://matrixhub.ai',
  baseUrl: '/',

  organizationName: 'matrixhub-ai',
  projectName: 'matrixhub',

  onBrokenLinks: 'warn',
  onBrokenAnchors: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'zh-CN'],
  },


  presets: [
    [
      'classic',
      {
        docs: {
          path: './docs',
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
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: true,
      respectPrefersColorScheme: false,
    },
    navbar: {
      title: 'MatrixHub',
      logo: {
        alt: 'MatrixHub Logo',
        src: 'img/matrixhub-logo.png',
        style: { height: '32px', width: '32px', borderRadius: '6px' },
      },
      hideOnScroll: false,
      items: [
        {
          to: 'docs/intro',
          label: 'Documentation',
          position: 'left',
        },
        {
          href: 'https://github.com/matrixhub-ai/matrixhub',
          label: 'GitHub',
          position: 'right',
          className: 'navbar-github-link',
        },
        {
          href: 'https://github.com/matrixhub-ai/matrixhub',
          label: 'Get Started',
          position: 'right',
          className: 'navbar-get-started-button',
        },
        {
          type: 'localeDropdown',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Product',
          items: [
            { label: 'Integrations', href: '#' },
            { label: 'Roadmap', href: '#' },
          ],
        },
        {
          title: 'Resources',
          items: [
            { label: 'Documentation', href: 'https://github.com/matrixhub-ai/matrixhub' },
            { label: 'API Reference', href: '#' },
            { label: 'Community', href: '#' },
          ],
        },
        {
          title: 'Connect',
          items: [
            { label: 'GitHub', href: 'https://github.com/matrixhub-ai/matrixhub' },
          ],
        },
      ],
      copyright: `© ${new Date().getFullYear()} MatrixHub Open Source. Apache 2.0 Licensed.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
