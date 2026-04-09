import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  tutorialSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/quickstart',
        'getting-started/examples',
      ],
    },
    {
      type: 'category',
      label: 'Operations',
      items: [
        'operations/overview',
        {
          type: 'category',
          label: 'Project Management',
          items: [
            'operations/projects/create-delete',
            'operations/projects/members',
          ],
        },
        {
          type: 'category',
          label: 'Models and Datasets',
          items: [
            'operations/assets/create-delete',
            'operations/assets/upload-download',
            'operations/assets/popular',
          ],
        },
        {
          type: 'category',
          label: 'Repository Management',
          items: [
            'operations/repository/management',
          ],
        },
        {
          type: 'category',
          label: 'User Management',
          items: [
            'operations/users/management',
          ],
        },
        {
          type: 'category',
          label: 'Remote Sync',
          items: [
            'operations/sync/remote',
          ],
        },
        {
          type: 'category',
          label: 'Developer',
          items: [
            'operations/developer/access-token',
          ],
        },
      ],
    },
  ],
};

export default sidebars;
