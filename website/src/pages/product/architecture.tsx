import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function Architecture(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'architecture.title', message: 'Architecture' })}
      description={translate({
        id: 'architecture.description',
        message:
          'Technical architecture of MatrixHub — proxy layer, caching engine, storage backends, and API surface.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="architecture.heading">Architecture</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="architecture.subtitle">
                Technical architecture of MatrixHub — proxy layer, caching engine, storage backends,
                and API surface.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-12">
          <div className="grid md:grid-cols-2 gap-8">
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <h2 className="text-2xl font-bold text-white mb-6 flex items-center gap-3">
                <span className="text-3xl">🎛️</span>
                <Translate id="architecture.controlPlane.title">Control Plane</Translate>
              </h2>
              <ul className="space-y-4 text-slate-400">
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.cp1">API gateway and routing</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.cp2">Authentication and authorization</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.cp3">Repository metadata management</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.cp4">Project, policy, quota, and audit services</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.cp5">Web UI and admin workflows</Translate></li>
              </ul>
            </div>

            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <h2 className="text-2xl font-bold text-white mb-6 flex items-center gap-3">
                <span className="text-3xl">🚀</span>
                <Translate id="architecture.dataPlane.title">Data Plane</Translate>
              </h2>
              <ul className="space-y-4 text-slate-400">
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.dp1">Large-file upload and download path</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.dp2">Cache population path</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.dp3">Replication path</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.dp4">Import and export path for air-gapped transfer</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.dp5">Acceleration path for direct-to-GPU loading</Translate></li>
              </ul>
            </div>
          </div>

          <div className="grid md:grid-cols-2 gap-8">
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <h2 className="text-2xl font-bold text-white mb-6 flex items-center gap-3">
                <span className="text-3xl">💾</span>
                <Translate id="architecture.storage.title">Storage & State</Translate>
              </h2>
              <ul className="space-y-4 text-slate-400">
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.store1">Object store or filesystem for large artifacts</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.store2">Relational database for metadata, tokens, tags</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.store3">Redis or in-memory cache for hot metadata</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.store4">Message queue for async tasks</Translate></li>
              </ul>
            </div>

            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <h2 className="text-2xl font-bold text-white mb-6 flex items-center gap-3">
                <span className="text-3xl">⚡</span>
                <Translate id="architecture.execution.title">Execution Model</Translate>
              </h2>
              <ul className="space-y-4 text-slate-400">
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.exec1">Stateless API services</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.exec2">Async workers for transfer orchestration</Translate></li>
                <li className="flex items-center gap-3"><span className="text-green-500">■</span> <Translate id="architecture.exec3">Explicit retry, backoff, and dead-letter handling</Translate></li>
              </ul>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
