import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

const sections = [
  {
    emoji: '⭐',
    title: 'GitHub',
    description:
      'Star and follow the MatrixHub repository for updates, issues, and releases.',
    linkText: 'Visit Repository →',
    href: 'https://github.com/matrixhub',
  },
  {
    emoji: '🤝',
    title: 'Contributing',
    description:
      'We welcome contributions of all kinds — code, documentation, bug reports, and feature requests.',
    linkText: 'Contribution Guide →',
    href: '/docs/contributing',
  },
  {
    emoji: '💬',
    title: 'Community Channels',
    description:
      'Join the conversation on Discord, GitHub Discussions, or our mailing list.',
    linkText: 'Join Discord →',
    href: 'https://discord.gg/matrixhub',
  },
];

export default function Community(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'community.title', message: 'Community' })}
      description={translate({
        id: 'community.description',
        message:
          'Join the MatrixHub community — contribute, discuss, and build together.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        {/* Hero */}
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="community.heading">Community</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="community.subtitle">
                Join the MatrixHub community — contribute, discuss, and build together.
              </Translate>
            </p>
          </div>
        </section>

        {/* Community sections */}
        <section className="py-16 max-w-4xl mx-auto px-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {sections.map((section) => (
              <div
                key={section.title}
                className="rounded-xl border border-slate-800 bg-slate-900/60 p-8 text-center"
              >
                <div className="text-4xl mb-4">{section.emoji}</div>
                <h3 className="text-xl font-semibold text-white mb-3">{section.title}</h3>
                <p className="text-sm text-slate-400 mb-6">{section.description}</p>
                <a
                  href={section.href}
                  className="text-sm font-medium text-green-400 hover:text-green-300 transition no-underline"
                >
                  {section.linkText}
                </a>
              </div>
            ))}
          </div>
        </section>
      </main>
    </Layout>
  );
}
