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
      label: 'Operations',
      items: [
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
          label: 'Model Repository',
          items: [
            'operations/model-repo/upload-download',
            'operations/model-repo/project-setting',
          ],
        },
        {
          type: 'category',
          label: 'Profile',
          items: [
            'operations/profile/access-token',
          ],
        },
      ],
    },
  ],
};

export default sidebars;
