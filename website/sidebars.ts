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
      label: '快速开始',
      items: [
        'getting-started/quickstart',
        'getting-started/examples',
      ],
    },
    {
      type: 'category',
      label: '操作指南',
      items: [
        'operations/overview',
        {
          type: 'category',
          label: '模型仓库',
          items: [
            'operations/model-repo/upload-download',
            'operations/model-repo/popular',
          ],
        },
        {
          type: 'category',
          label: '数据集',
          items: [
            'operations/datasets/create-delete',
          ],
        },
        {
          type: 'category',
          label: '项目管理',
          items: [
            'operations/project-management/create-delete',
            'operations/project-management/members',
          ],
        },
        {
          type: 'category',
          label: '个人中心',
          items: [
            'operations/profile/access-token',
          ],
        },
        {
          type: 'category',
          label: '平台设置',
          items: [
            {
              type: 'category',
              label: '用户管理',
              items: [
                'operations/platform-settings/user-management',
              ],
            },
            {
              type: 'category',
              label: '仓库管理',
              items: [
                'operations/platform-settings/repository-management',
              ],
            },
            {
              type: 'category',
              label: '远程同步',
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
