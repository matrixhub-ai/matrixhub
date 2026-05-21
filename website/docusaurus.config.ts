import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'MatrixHub',
  tagline: 'The Open Source Hub for AI Models',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://matrixhub.ai',
  baseUrl: '/',

  organizationName: 'matrixhub-ai',
  projectName: 'matrixhub',

  onBrokenLinks: 'warn',
  onBrokenAnchors: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'zh-CN'],
    localeConfigs: {
      en: {
        label: 'English',
      },
      'zh-CN': {
        label: '简体中文',
      },
    },
  },

  presets: [
    [
      'classic',
      {
        docs: {
          path: './docs',
          routeBasePath: 'docs',
          sidebarPath: './sidebars.ts',
        },
        blog: {
          routeBasePath: 'blog',
          showReadingTime: true,
        },
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
        src: 'img/matrixhub-icon-colorlight.png',
        style: { height: '32px', width: '32px', borderRadius: '6px' },
      },
      hideOnScroll: false,
      items: [
        {
          to: '/product',
          label: 'Product',
          position: 'left',
        },
        {
          to: '/docs/overview',
          label: 'Docs',
          position: 'left',
        },
        {
          to: '/blog',
          label: 'Blog',
          position: 'left',
        },
        {
          to: '/community',
          label: 'Community',
          position: 'left',
        },
        {
          type: 'localeDropdown',
          position: 'right',
        },
        {
          href: 'https://github.com/matrixhub-ai/matrixhub',
          label: 'GitHub',
          position: 'right',
          className: 'navbar-github-link',
        },
        {
          to: '/docs/getting-started',
          label: 'Get Started',
          position: 'right',
          className: 'navbar-get-started-button',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Product',
          items: [
            { label: 'Why MatrixHub', to: '/product/why-matrixhub' },
            { label: 'Use Cases', to: '/product/use-cases' },
            { label: 'Architecture', to: '/product/architecture' },
            { label: 'Comparison', to: '/product/comparison' },
          ],
        },
        {
          title: 'Resources',
          items: [
            { label: 'Documentation', to: '/docs/overview' },
            { label: 'API Reference', to: '/docs/reference' },
            { label: 'Blog', to: '/blog' },
            { label: 'Changelog', to: '/changelog' },
          ],
        },
        {
          title: 'Community',
          items: [
            { label: 'GitHub', href: 'https://github.com/matrixhub-ai/matrixhub' },
            { label: 'Community', to: '/community' },
            { label: 'Roadmap', to: '/roadmap' },
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
