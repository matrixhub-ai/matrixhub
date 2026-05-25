import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';

const features = [
  {
    title: 'Why MatrixHub',
    description: 'Why organizations choose MatrixHub as their private model registry.',
    href: '/product/why-matrixhub',
    emoji: '🎯',
  },
  {
    title: 'Use Cases',
    description: 'Real-world deployment scenarios across enterprise AI teams.',
    href: '/product/use-cases',
    emoji: '💼',
  },
  {
    title: 'Architecture',
    description: 'Proxy layer, caching engine, storage backends, and API surface.',
    href: '/product/architecture',
    emoji: '🏗️',
  },
  {
    title: 'Hugging Face Compatible',
    description: 'Drop-in compatibility with the Hugging Face ecosystem.',
    href: '/product/huggingface-compatible',
    emoji: '🤗',
  },
  {
    title: 'Inference Acceleration',
    description: 'Accelerate model distribution for vLLM, SGLang, and more.',
    href: '/product/inference-acceleration',
    emoji: '⚡',
  },
  {
    title: 'Air-Gapped Delivery',
    description: 'Securely deliver models to isolated environments.',
    href: '/product/air-gapped-delivery',
    emoji: '🔒',
  },
  {
    title: 'Remote Sync',
    description: 'Synchronize models across data centers and regions.',
    href: '/product/remote-sync',
    emoji: '🔄',
  },
  {
    title: 'Governance',
    description: 'RBAC, audit trails, and compliance controls for enterprise AI.',
    href: '/product/governance',
    emoji: '🛡️',
  },
];

export default function ProductOverview(): React.ReactElement {
  return (
    <Layout
      title={translate({ id: 'product.title', message: 'Product' })}
      description={translate({
        id: 'product.description',
        message:
          'Explore MatrixHub capabilities — the open-source, self-hosted model hub built for enterprise inference at scale.',
      })}
    >
      <main className="bg-[#0d1117] text-slate-300 min-h-screen">
        {/* Hero */}
        <section className="py-20 relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-green-500/10 blur-[120px] rounded-full pointer-events-none"></div>
          <div className="max-w-4xl mx-auto px-6 relative z-10 text-center">
            <h1 className="text-4xl sm:text-5xl font-extrabold text-white mb-6">
              <Translate id="product.heading">Product</Translate>
            </h1>
            <p className="text-lg text-slate-400 max-w-2xl mx-auto">
              <Translate id="product.subtitle">
                Explore MatrixHub capabilities — the open-source, self-hosted model hub built for
                enterprise inference at scale.
              </Translate>
            </p>
          </div>
        </section>

        {/* Feature cards grid */}
        <section className="py-16 max-w-6xl mx-auto px-6">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {features.map((feature) => (
              <a
                key={feature.href}
                href={feature.href}
                className="group block rounded-xl border border-slate-800 bg-slate-900/60 p-6 transition hover:border-green-500/40 hover:bg-slate-800/60 no-underline"
              >
                <div className="text-3xl mb-4">{feature.emoji}</div>
                <h3 className="text-lg font-semibold text-white mb-2 group-hover:text-green-400 transition">
                  {feature.title}
                </h3>
                <p className="text-sm text-slate-400">{feature.description}</p>
              </a>
            ))}
          </div>
        </section>

        {/* Comparison CTA */}
        <section className="py-16 max-w-4xl mx-auto px-6 text-center">
          <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-10">
            <h2 className="text-2xl font-bold text-white mb-4">
              <Translate id="product.comparison.heading">How does MatrixHub compare?</Translate>
            </h2>
            <p className="text-slate-400 mb-6">
              <Translate id="product.comparison.subtitle">
                See how MatrixHub stacks up against Hugging Face, Harbor, and other model management
                solutions.
              </Translate>
            </p>
            <a
              href="/product/comparison"
              className="inline-block rounded-lg bg-green-600 px-6 py-3 text-sm font-semibold text-white hover:bg-green-500 transition no-underline"
            >
              <Translate id="product.comparison.cta">View Comparison</Translate>
            </a>
          </div>
        </section>
      </main>
    </Layout>
  );
}
