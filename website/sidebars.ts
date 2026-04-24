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
          label: 'Model Repository',
          items: [
            'operations/model-repo/upload-download',
            'operations/model-repo/project-setting',
          ],
        },
        {
          type: 'category',
          label: 'Datasets',
          items: [
            'operations/datasets/create-delete',
          ],
        },
        {
          type: 'category',
          label: 'Project Management',
          items: [
            'operations/project-management/create-delete',
            'operations/project-management/members',
          ],
        },
        {
          type: 'category',
          label: 'Profile',
          items: [
            'operations/profile/access-token',
          ],
        },
        {
          type: 'category',
          label: 'Platform Settings',
          items: [
            {
              type: 'category',
              label: 'User Management',
              items: [
                'operations/platform-settings/user-management',
              ],
            },
            {
              type: 'category',
              label: 'Repository Management',
              items: [
                'operations/platform-settings/repository-management',
              ],
            },
            {
              type: 'category',
              label: 'Remote Sync',
              items: [
                'operations/platform-settings/remote-sync',
              ],
            },
          ],
        },
      ],
    },
  ],
};

export default sidebars;
