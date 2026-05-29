import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

export default function Governance(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'governance.title', message: 'Governance' })}
      description={translate({
        id: 'governance.description',
        message: 'RBAC, audit trails, and compliance controls for enterprise AI.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="governance.heading">Governance</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="governance.subtitle">
                RBAC, audit trails, and compliance controls for enterprise AI.
              </Translate>
            </p>
          </div>
        </section>

        <section className="py-16 max-w-5xl mx-auto px-6 space-y-16">
          <div className="text-center max-w-3xl mx-auto mb-12">
            <h2 className="text-3xl font-bold text-white mb-6">
              <Translate id="governance.artifact.title">Enterprise Artifact Management</Translate>
            </h2>
            <p className="text-slate-400 text-lg leading-relaxed">
              <Translate id="governance.artifact.desc">
                Multiple internal teams fine-tune models. Versions drift between training, testing, and production. MatrixHub turns models into governed production artifacts instead of informal file shares.
              </Translate>
            </p>
          </div>

          <div className="grid md:grid-cols-3 gap-6">
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <div className="text-4xl mb-4">👥</div>
              <h3 className="text-xl font-bold text-white mb-3">
                <Translate id="governance.rbac.title">RBAC & Multi-Tenancy</Translate>
              </h3>
              <p className="text-slate-400">
                <Translate id="governance.rbac.desc">
                  Project-based isolation with granular permissions. Integrate seamlessly with your existing LDAP/SSO providers for centralized access control.
                </Translate>
              </p>
            </div>
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <div className="text-4xl mb-4">📝</div>
              <h3 className="text-xl font-bold text-white mb-3">
                <Translate id="governance.audit.title">Audit & Compliance</Translate>
              </h3>
              <p className="text-slate-400">
                <Translate id="governance.audit.desc">
                  Full traceability with comprehensive logs. Track who uploaded, promoted, downloaded, or changed access rules for every model version.
                </Translate>
              </p>
            </div>
            <div className="bg-slate-800/30 p-8 rounded-3xl border border-slate-700/50 hover:border-slate-600/50 transition-colors">
              <div className="text-4xl mb-4">🛡️</div>
              <h3 className="text-xl font-bold text-white mb-3">
                <Translate id="governance.integrity.title">Integrity Protection</Translate>
              </h3>
              <p className="text-slate-400">
                <Translate id="governance.integrity.desc">
                  Tag locking, release promotion workflows, built-in malware scanning, and content signing to ensure your production models remain untampered.
                </Translate>
              </p>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
