import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function WhyMatrixHub(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'whyMatrixhub.title', message: 'Why MatrixHub' })}
      description={translate({
        id: 'whyMatrixhub.description',
        message:
          'Why organizations choose MatrixHub as their private model registry for enterprise AI.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="whyMatrixhub.heading">Why MatrixHub</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="whyMatrixhub.subtitle">
                Why organizations choose MatrixHub as their private model registry for enterprise AI.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-20">
          <div>
            <h2 className="text-3xl font-bold text-white mb-8 border-b border-slate-800 pb-4">
              <Translate id="why.problem.title">The Challenge of Enterprise AI</Translate>
            </h2>
            <p className="text-slate-400 mb-8 text-lg">
              <Translate id="why.problem.desc">
                Enterprise inference environments have a different set of constraints than public model-sharing platforms. Common pain points include:
              </Translate>
            </p>
            <div className="grid md:grid-cols-2 gap-6">
              <div className="bg-slate-800/50 p-6 rounded-2xl border border-slate-700/50 hover:border-green-500/30 transition-colors">
                <div className="text-3xl mb-4">🐢</div>
                <h3 className="text-xl font-bold text-white mb-2">
                  <Translate id="why.pain1.title">Bandwidth Bottlenecks</Translate>
                </h3>
                <p className="text-slate-400">
                  <Translate id="why.pain1.desc">Large models are expensive and slow to download repeatedly from public endpoints. Production clusters may include dozens or hundreds of nodes starting at the same time.</Translate>
                </p>
              </div>
              <div className="bg-slate-800/50 p-6 rounded-2xl border border-slate-700/50 hover:border-green-500/30 transition-colors">
                <div className="text-3xl mb-4">🔒</div>
                <h3 className="text-xl font-bold text-white mb-2">
                  <Translate id="why.pain2.title">Network Isolation</Translate>
                </h3>
                <p className="text-slate-400">
                  <Translate id="why.pain2.desc">Air-gapped and regulated environments need controlled model import and export, breaking standard workflows.</Translate>
                </p>
              </div>
              <div className="bg-slate-800/50 p-6 rounded-2xl border border-slate-700/50 hover:border-green-500/30 transition-colors">
                <div className="text-3xl mb-4">⚠️</div>
                <h3 className="text-xl font-bold text-white mb-2">
                  <Translate id="why.pain3.title">Model Version Drift</Translate>
                </h3>
                <p className="text-slate-400">
                  <Translate id="why.pain3.desc">Fine-tuned model versions become inconsistent across training, testing, and production without formal artifacts.</Translate>
                </p>
              </div>
              <div className="bg-slate-800/50 p-6 rounded-2xl border border-slate-700/50 hover:border-green-500/30 transition-colors">
                <div className="text-3xl mb-4">🌍</div>
                <h3 className="text-xl font-bold text-white mb-2">
                  <Translate id="why.pain4.title">Cross-Region Latency</Translate>
                </h3>
                <p className="text-slate-400">
                  <Translate id="why.pain4.desc">Replicating tens of TBs over unstable WAN links is slow and painful. Global teams need local access.</Translate>
                </p>
              </div>
            </div>
          </div>

          <div>
            <h2 className="text-3xl font-bold text-white mb-8 border-b border-slate-800 pb-4">
              <Translate id="why.positioning.title">Our Product Thesis</Translate>
            </h2>
            <div className="space-y-6">
              <div className="bg-gradient-to-r from-slate-800/80 to-slate-800/30 p-8 rounded-3xl border border-slate-700/50 flex flex-col md:flex-row gap-6 items-start">
                <div className="bg-slate-900 p-4 rounded-2xl text-green-400 text-3xl shrink-0 border border-slate-700/50">🏢</div>
                <div>
                  <h3 className="text-2xl font-bold text-white mb-2">
                    <Translate id="why.pos1.title">Private Deployment First</Translate>
                  </h3>
                  <p className="text-slate-400 leading-relaxed">
                    <Translate id="why.pos1.desc">
                      Compared with public model hubs, MatrixHub is private-deployment-first and enterprise-operations-first. We optimize for internal clusters and controlled networks.
                    </Translate>
                  </p>
                </div>
              </div>

              <div className="bg-gradient-to-r from-slate-800/80 to-slate-800/30 p-8 rounded-3xl border border-slate-700/50 flex flex-col md:flex-row gap-6 items-start">
                <div className="bg-slate-900 p-4 rounded-2xl text-green-400 text-3xl shrink-0 border border-slate-700/50">🤗</div>
                <div>
                  <h3 className="text-2xl font-bold text-white mb-2">
                    <Translate id="why.pos2.title">Hugging Face Compatible</Translate>
                  </h3>
                  <p className="text-slate-400 leading-relaxed">
                    <Translate id="why.pos2.desc">
                      Compared with general artifact registries, MatrixHub is natively optimized for Hugging Face-compatible model access and large-model distribution semantics.
                    </Translate>
                  </p>
                </div>
              </div>

              <div className="bg-gradient-to-r from-slate-800/80 to-slate-800/30 p-8 rounded-3xl border border-slate-700/50 flex flex-col md:flex-row gap-6 items-start">
                <div className="bg-slate-900 p-4 rounded-2xl text-green-400 text-3xl shrink-0 border border-slate-700/50">⚙️</div>
                <div>
                  <h3 className="text-2xl font-bold text-white mb-2">
                    <Translate id="why.pos3.title">Operational Reliability</Translate>
                  </h3>
                  <p className="text-slate-400 leading-relaxed">
                    <Translate id="why.pos3.desc">
                      We focus on completeness, operational reliability, and inference-oriented workflows rather than on breadth of unrelated features.
                    </Translate>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
