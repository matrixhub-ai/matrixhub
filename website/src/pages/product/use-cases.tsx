import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function UseCases(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'useCases.title', message: 'Use Cases' })}
      description={translate({
        id: 'useCases.description',
        message:
          'Real-world deployment scenarios for MatrixHub across enterprise AI teams.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="useCases.heading">Use Cases</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="useCases.subtitle">
                Real-world deployment scenarios for MatrixHub across enterprise AI teams.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-12">
          {/* Scenario 1 */}
          <div className="flex flex-col md:flex-row gap-8 items-start bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
            <div className="md:w-1/3 shrink-0">
              <div className="text-sm font-bold text-green-500 mb-2 tracking-wider uppercase">
                <Translate id="useCases.uc1.label">Scenario 1</Translate>
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">
                <Translate id="useCases.uc1.title">Intranet Inference Acceleration</Translate>
              </h2>
            </div>
            <div className="md:w-2/3 space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-2">
                  <Translate id="useCases.uc1.scenarioLabel">The Scenario</Translate>
                </h3>
                <p className="text-slate-400 leading-relaxed">
                  <Translate id="useCases.uc1.scenario">An internal production environment runs a vLLM cluster with 100 GPU servers. A 70B model may exceed 130 GB. If every node pulls from the public Hugging Face endpoint independently, startup becomes slow and public bandwidth becomes a bottleneck.</Translate>
                </p>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-3">
                  <Translate id="useCases.uc1.valueLabel">Product Value</Translate>
                </h3>
                <ul className="space-y-2">
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc1.value.0">Reduce cold-start amplification</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc1.value.1">Reduce external bandwidth pressure</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc1.value.2">Provide a single operational control point for model access</Translate></span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          {/* Scenario 2 */}
          <div className="flex flex-col md:flex-row gap-8 items-start bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
            <div className="md:w-1/3 shrink-0">
              <div className="text-sm font-bold text-green-500 mb-2 tracking-wider uppercase">
                <Translate id="useCases.uc2.label">Scenario 2</Translate>
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">
                <Translate id="useCases.uc2.title">Air-Gapped Model Transfer</Translate>
              </h2>
            </div>
            <div className="md:w-2/3 space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-2">
                  <Translate id="useCases.uc2.scenarioLabel">The Scenario</Translate>
                </h3>
                <p className="text-slate-400 leading-relaxed">
                  <Translate id="useCases.uc2.scenario">Government, defense, or core financial environments need access to open models but operate in isolated networks.</Translate>
                </p>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-3">
                  <Translate id="useCases.uc2.valueLabel">Product Value</Translate>
                </h3>
                <ul className="space-y-2">
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc2.value.0">Preserve air-gap boundaries</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc2.value.1">Reduce manual model handling overhead</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc2.value.2">Keep client workflows consistent across connected and disconnected environments</Translate></span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          {/* Scenario 3 */}
          <div className="flex flex-col md:flex-row gap-8 items-start bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
            <div className="md:w-1/3 shrink-0">
              <div className="text-sm font-bold text-green-500 mb-2 tracking-wider uppercase">
                <Translate id="useCases.uc3.label">Scenario 3</Translate>
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">
                <Translate id="useCases.uc3.title">Enterprise Model Artifact Management</Translate>
              </h2>
            </div>
            <div className="md:w-2/3 space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-2">
                  <Translate id="useCases.uc3.scenarioLabel">The Scenario</Translate>
                </h3>
                <p className="text-slate-400 leading-relaxed">
                  <Translate id="useCases.uc3.scenario">Multiple internal teams fine-tune models. Versions drift between training, testing, and production. Operators need fixed, governed releases instead of informal file sharing.</Translate>
                </p>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-3">
                  <Translate id="useCases.uc3.valueLabel">Product Value</Translate>
                </h3>
                <ul className="space-y-2">
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc3.value.0">Turn models into governed production artifacts</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc3.value.1">Improve release reproducibility</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc3.value.2">Reduce accidental model drift across environments</Translate></span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          {/* Scenario 4 */}
          <div className="flex flex-col md:flex-row gap-8 items-start bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
            <div className="md:w-1/3 shrink-0">
              <div className="text-sm font-bold text-green-500 mb-2 tracking-wider uppercase">
                <Translate id="useCases.uc4.label">Scenario 4</Translate>
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">
                <Translate id="useCases.uc4.title">Cross-Region Distribution</Translate>
              </h2>
            </div>
            <div className="md:w-2/3 space-y-6">
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-2">
                  <Translate id="useCases.uc4.scenarioLabel">The Scenario</Translate>
                </h3>
                <p className="text-slate-400 leading-relaxed">
                  <Translate id="useCases.uc4.scenario">Global teams operate compute centers in different regions. Replicating tens of TB of weights and datasets over unstable WAN links is slow and operationally painful.</Translate>
                </p>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-slate-200 mb-3">
                  <Translate id="useCases.uc4.valueLabel">Product Value</Translate>
                </h3>
                <ul className="space-y-2">
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc4.value.0">Lower cross-region latency</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc4.value.1">Reduce WAN congestion</Translate></span>
                  </li>
                  <li className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-green-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                    <span className="text-slate-300"><Translate id="useCases.uc4.value.2">Make global collaboration operationally manageable</Translate></span>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
