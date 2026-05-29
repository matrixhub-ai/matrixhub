import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function AirGappedDelivery(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'airgap.title', message: 'Air-Gapped Delivery' })}
      description={translate({
        id: 'airgap.description',
        message: 'Securely deliver models to isolated environments.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="airgap.heading">Air-Gapped Delivery</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="airgap.subtitle">
                Securely deliver models to isolated environments.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-16">
          <div className="text-center max-w-3xl mx-auto mb-12">
            <h2 className="text-3xl font-bold text-white mb-6">
              <Translate id="airgap.isolated.title">Seamless Offline Experience</Translate>
            </h2>
            <p className="text-slate-400 text-lg leading-relaxed">
              <Translate id="airgap.isolated.desc">
                Securely ferry models into isolated networks while maintaining a native Hugging Face experience for researchers—no internet required.
              </Translate>
            </p>
          </div>

          <div className="bg-slate-800/30 p-10 rounded-3xl border border-slate-700/50 relative overflow-hidden">
            <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-transparent via-green-500 to-transparent opacity-50"></div>
            <h3 className="text-2xl font-bold text-white mb-10 text-center">
              <Translate id="airgap.workflow.title">The Air-Gapped Workflow</Translate>
            </h3>
            
            <div className="space-y-8 relative before:absolute before:inset-0 before:ml-6 before:-translate-x-px md:before:mx-auto md:before:translate-x-0 before:h-full before:w-0.5 before:bg-gradient-to-b before:from-transparent before:via-slate-700 before:to-transparent">
              {/* Step 1 */}
              <div className="relative flex items-center justify-between md:justify-normal md:flex-row-reverse group">
                <div className="flex items-center justify-center w-12 h-12 rounded-full border-4 border-slate-900 bg-green-500 text-slate-900 font-bold text-lg shrink-0 md:order-1 md:-translate-x-1/2 shadow-[0_0_0_4px_rgba(34,197,94,0.2)] z-10">
                  1
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-3rem)] bg-slate-900 p-6 rounded-2xl border border-slate-700/50 shadow-xl hover:border-green-500/30 transition-colors">
                  <h4 className="text-xl font-bold text-white mb-2">
                    <Translate id="airgap.step1.title">Mirror in DMZ</Translate>
                  </h4>
                  <p className="text-slate-400">
                    <Translate id="airgap.step1.desc">An administrator deploys MatrixHub in a connected staging or DMZ environment and mirrors required models into the staging hub.</Translate>
                  </p>
                </div>
              </div>
              
              {/* Step 2 */}
              <div className="relative flex items-center justify-between md:justify-normal md:flex-row group">
                <div className="flex items-center justify-center w-12 h-12 rounded-full border-4 border-slate-900 bg-green-500 text-slate-900 font-bold text-lg shrink-0 md:order-1 md:translate-x-1/2 shadow-[0_0_0_4px_rgba(34,197,94,0.2)] z-10">
                  2
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-3rem)] bg-slate-900 p-6 rounded-2xl border border-slate-700/50 shadow-xl hover:border-green-500/30 transition-colors">
                  <h4 className="text-xl font-bold text-white mb-2">
                    <Translate id="airgap.step2.title">Export & Transfer</Translate>
                  </h4>
                  <p className="text-slate-400">
                    <Translate id="airgap.step2.desc">The mirrored content is exported into portable archives and securely transferred into the isolated production network.</Translate>
                  </p>
                </div>
              </div>

              {/* Step 3 */}
              <div className="relative flex items-center justify-between md:justify-normal md:flex-row-reverse group">
                <div className="flex items-center justify-center w-12 h-12 rounded-full border-4 border-slate-900 bg-green-500 text-slate-900 font-bold text-lg shrink-0 md:order-1 md:-translate-x-1/2 shadow-[0_0_0_4px_rgba(34,197,94,0.2)] z-10">
                  3
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-3rem)] bg-slate-900 p-6 rounded-2xl border border-slate-700/50 shadow-xl hover:border-green-500/30 transition-colors">
                  <h4 className="text-xl font-bold text-white mb-2">
                    <Translate id="airgap.step3.title">Import to Production</Translate>
                  </h4>
                  <p className="text-slate-400">
                    <Translate id="airgap.step3.desc">The production hub imports the content. Models are automatically parsed, checksummed, and made available.</Translate>
                  </p>
                </div>
              </div>

              {/* Step 4 */}
              <div className="relative flex items-center justify-between md:justify-normal md:flex-row group">
                <div className="flex items-center justify-center w-12 h-12 rounded-full border-4 border-slate-900 bg-green-500 text-slate-900 font-bold text-lg shrink-0 md:order-1 md:translate-x-1/2 shadow-[0_0_0_4px_rgba(34,197,94,0.2)] z-10">
                  4
                </div>
                <div className="w-[calc(100%-4rem)] md:w-[calc(50%-3rem)] bg-slate-900 p-6 rounded-2xl border border-slate-700/50 shadow-xl hover:border-green-500/30 transition-colors">
                  <h4 className="text-xl font-bold text-white mb-2">
                    <Translate id="airgap.step4.title">Native Access</Translate>
                  </h4>
                  <p className="text-slate-400">
                    <Translate id="airgap.step4.desc">Internal users access models through the exact same Hugging Face-compatible interface without any internet access.</Translate>
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
