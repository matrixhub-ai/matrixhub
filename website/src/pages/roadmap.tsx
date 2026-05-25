import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function Roadmap(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'roadmap.title', message: 'Roadmap' })}
      description={translate({
        id: 'roadmap.description',
        message:
          'See what is planned for MatrixHub — upcoming features, improvements, and milestones.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="roadmap.heading">Roadmap</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="roadmap.subtitle">
                See what is planned for MatrixHub — upcoming features, improvements, and milestones.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-4xl mx-auto px-6">
          <div className="text-center text-slate-500">
            <p>🚧 This page is under construction. Content coming soon.</p>
          </div>
        </section>
      </main>
    </Layout>
  );
}
