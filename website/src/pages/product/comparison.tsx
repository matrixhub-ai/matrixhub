import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function Comparison(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'comparison.title', message: 'Comparison' })}
      description={translate({
        id: 'comparison.description',
        message:
          'How MatrixHub compares to Hugging Face, Harbor, and other model management solutions.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="comparison.heading">Comparison</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="comparison.subtitle">
                How MatrixHub compares to Hugging Face, Harbor, and other model management solutions.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-16">
          {/* Positioning */}
          <div className="text-center max-w-3xl mx-auto mb-4">
            <p className="text-slate-400 text-lg leading-relaxed">
              <Translate id="comparison.positioning">
                MatrixHub sits between public model hubs and general-purpose artifact registries. It focuses on the infrastructure layer that enterprise inference teams actually need: private deployment, fast distribution, release management, replication, and policy control.
              </Translate>
            </p>
          </div>

          {/* vs Public SaaS Model Hubs */}
          <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50">
            <div className="flex items-center gap-4 mb-6">
              <div className="bg-slate-900 p-3 rounded-2xl text-3xl border border-slate-700/50">🌐</div>
              <h2 className="text-2xl font-bold text-white">
                <Translate id="comparison.vsSaas.title">Against Public SaaS Model Hubs</Translate>
              </h2>
            </div>
            <p className="text-slate-400 mb-6 leading-relaxed">
              <Translate id="comparison.vsSaas.desc">
                Compared with public model hubs, MatrixHub is private-deployment-first and enterprise-operations-first.
              </Translate>
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsSaas.point1">Private deployment — optimized for internal clusters and controlled networks</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsSaas.point2">Governance and release controls designed for enterprise operations</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsSaas.point3">Replication and caching designed around infrastructure ownership</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsSaas.point4">Air-gapped delivery for isolated and regulated environments</Translate></span>
              </div>
            </div>
          </div>

          {/* vs General Artifact Registries */}
          <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50">
            <div className="flex items-center gap-4 mb-6">
              <div className="bg-slate-900 p-3 rounded-2xl text-3xl border border-slate-700/50">📦</div>
              <h2 className="text-2xl font-bold text-white">
                <Translate id="comparison.vsRegistry.title">Against General Artifact Registries</Translate>
              </h2>
            </div>
            <p className="text-slate-400 mb-6 leading-relaxed">
              <Translate id="comparison.vsRegistry.desc">
                Existing registries provide governance but are not optimized for Hugging Face-compatible model consumption. MatrixHub is optimized for Hugging Face-compatible model access and large-model distribution.
              </Translate>
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsRegistry.point1">Hugging Face-compatible access patterns — switch with zero code changes by redirecting HF_ENDPOINT</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsRegistry.point2">Model-aware caching and distribution workflows</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsRegistry.point3">Optimization specifically for vLLM and SGLang inference engines</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsRegistry.point4">Optimized for very large artifacts and repeated access patterns</Translate></span>
              </div>
            </div>
          </div>

          {/* vs Other Open-Source Model Hubs */}
          <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50">
            <div className="flex items-center gap-4 mb-6">
              <div className="bg-slate-900 p-3 rounded-2xl text-3xl border border-slate-700/50">🔓</div>
              <h2 className="text-2xl font-bold text-white">
                <Translate id="comparison.vsOSS.title">Against Other Open-Source Model Hubs</Translate>
              </h2>
            </div>
            <p className="text-slate-400 mb-6 leading-relaxed">
              <Translate id="comparison.vsOSS.desc">
                MatrixHub differentiates on operational completeness. The four key use cases — intranet acceleration, air-gapped transfer, governed model releases, and cross-region distribution — are the evaluation bar. If an alternative cannot handle those workflows end-to-end, it is not a full solution for the target user.
              </Translate>
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsOSS.point1">Inference-first, not training-platform-first</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsOSS.point2">Prefer simple, reliable workflows over broad but shallow functionality</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsOSS.point3">Complete end-to-end coverage of enterprise inference workflows</Translate></span>
              </div>
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                <span className="text-slate-300"><Translate id="comparison.vsOSS.point4">Default to private deployment and enterprise controls</Translate></span>
              </div>
            </div>
          </div>

        </section>
      </main>
    </Layout>
  );
}
