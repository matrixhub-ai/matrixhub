import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function RemoteSync(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'sync.title', message: 'Remote Sync' })}
      description={translate({
        id: 'sync.description',
        message: 'Synchronize models across data centers and regions.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="sync.heading">Remote Sync</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="sync.subtitle">
                Synchronize models across data centers and regions.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-16">
          <div className="text-center max-w-3xl mx-auto mb-12">
            <h2 className="text-3xl font-bold text-white mb-6">
              <Translate id="sync.global.title">Global Multi-Region Replication</Translate>
            </h2>
            <p className="text-slate-400 text-lg leading-relaxed">
              <Translate id="sync.global.desc">
                Automate asynchronous, resumable replication between data centers for high availability and low-latency local access. MatrixHub makes global collaboration operationally manageable.
              </Translate>
            </p>
          </div>

          <div className="grid md:grid-cols-2 gap-8">
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <div className="text-4xl mb-4">🔁</div>
              <h3 className="text-xl font-bold text-white mb-3">
                <Translate id="sync.resume.title">Chunking & Resume</Translate>
              </h3>
              <p className="text-slate-400">
                <Translate id="sync.resume.desc">
                  Transfers use chunking, resume, and retry so that network instability over WAN links does not require restarting massive 100GB+ downloads from scratch.
                </Translate>
              </p>
            </div>
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <div className="text-4xl mb-4">⚙️</div>
              <h3 className="text-xl font-bold text-white mb-3">
                <Translate id="sync.policy.title">Policy-Driven</Translate>
              </h3>
              <p className="text-slate-400">
                <Translate id="sync.policy.desc">
                  Configure precise rules based on Source Repository, Resource Filters, Trigger Modes (Auto/Manual), and Bandwidth Limits to control synchronization behaviors.
                </Translate>
              </p>
            </div>
          </div>

          <div className="bg-slate-900 p-8 rounded-3xl border border-slate-700/50">
            <h3 className="text-2xl font-bold text-white mb-6">
              <Translate id="sync.features.title">Synchronization Features</Translate>
            </h3>
            <div className="space-y-4">
              <div className="flex gap-4">
                <div className="text-green-500 mt-1">✓</div>
                <div>
                  <h4 className="text-white font-bold"><Translate id="sync.f0.title">Bandwidth Throttling</Translate></h4>
                  <p className="text-slate-400 text-sm"><Translate id="sync.f0.desc">Controls the synchronization download bandwidth to avoid saturating your WAN connection during peak hours.</Translate></p>
                </div>
              </div>
              <div className="flex gap-4">
                <div className="text-green-500 mt-1">✓</div>
                <div>
                  <h4 className="text-white font-bold"><Translate id="sync.f1.title">Resource Filtering</Translate></h4>
                  <p className="text-slate-400 text-sm"><Translate id="sync.f1.desc">Match resources to be synchronized by specific names or wildcards (e.g., `llama-3*`).</Translate></p>
                </div>
              </div>
              <div className="flex gap-4">
                <div className="text-green-500 mt-1">✓</div>
                <div>
                  <h4 className="text-white font-bold"><Translate id="sync.f2.title">Automatic Overwrite Controls</Translate></h4>
                  <p className="text-slate-400 text-sm"><Translate id="sync.f2.desc">Skip existing files or enforce remote-to-local overwrite policies.</Translate></p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
