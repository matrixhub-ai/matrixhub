import React, { useState, useEffect, useRef } from 'react';
import Layout from '@theme/Layout';
import {
  Zap,
  Shield,
  Database,
  Network,
  ArrowRight,
  Github,
  Lock,
  Rocket,
  ShieldCheck,
  Tags,
  Globe
} from 'lucide-react';



// Intersection Observer Hook for scroll animations
const useIntersectionObserver = (options = {}) => {
  const ref = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) {
        setIsVisible(true);
        observer.disconnect();
      }
    }, { threshold: 0.1, ...options });

    if (ref.current) {
      observer.observe(ref.current);
    }

    return () => observer.disconnect();
  }, []);

  return [ref, isVisible] as const;
};

// Animated Section Wrapper
const AnimatedSection = ({ children, className = '', delay = 0 }: { children: React.ReactNode, className?: string, delay?: number }) => {
  const [ref, isVisible] = useIntersectionObserver();

  return (
    <div
      ref={ref}
      className={`transition-all duration-700 ease-out ${className} ${isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}`}
      style={{ transitionDelay: `${delay}ms` }}
    >
      {children}
    </div>
  );
};

// Helper Components
const FeatureCard = ({ icon, title, desc, index }: { icon: React.ReactNode, title: string, desc: string, index: number }) => {
  const [ref, isVisible] = useIntersectionObserver();

  return (
    <div
      ref={ref}
      className={`p-6 rounded-xl bg-[#161b22] border border-slate-800 hover:border-green-500/50 hover:shadow-[0_0_20px_rgba(34,197,94,0.1)] transition-all duration-500 group h-full hover-lift ${isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}`}
      style={{ transitionDelay: `${index * 100}ms` }}
    >
      <div className="w-12 h-12 bg-slate-800 rounded-lg flex items-center justify-center text-green-500 mb-4 group-hover:scale-110 group-hover:bg-green-500/20 transition-all duration-300">
        {icon}
      </div>
      <h3 className="text-xl font-bold text-white mb-3">{title}</h3>
      <p className="text-slate-400 leading-relaxed text-sm">{desc}</p>
    </div>
  );
};

const UseCaseCard = ({ icon, title, desc, index }: { icon: React.ReactNode, title: string, desc: string, index: number }) => {
  const [ref, isVisible] = useIntersectionObserver();

  return (
    <div
      ref={ref}
      className={`flex gap-4 transition-all duration-500 ${isVisible ? 'opacity-100 translate-x-0' : 'opacity-0 -translate-x-8'}`}
      style={{ transitionDelay: `${index * 150}ms` }}
    >
      <div className="flex-shrink-0 mt-1">
        <div className="w-10 h-10 rounded-full bg-green-500/20 flex items-center justify-center text-green-500 group-hover:bg-green-500/30 transition-colors">
          {icon}
        </div>
      </div>
      <div>
        <h3 className="text-lg font-bold text-white mb-2">{title}</h3>
        <p className="text-slate-400 text-sm leading-relaxed">{desc}</p>
      </div>
    </div>
  );
};

// Tech Partner Logo Item
const TechLogo = ({ logo, name, index }: { logo: React.ReactNode, name: string, index: number }) => {
  const [ref, isVisible] = useIntersectionObserver();

  return (
    <div
      ref={ref}
      className={`flex items-center gap-2 sm:gap-3 text-lg sm:text-xl font-bold text-white logo-hover cursor-pointer transition-all duration-500 ${isVisible ? 'opacity-100 scale-100' : 'opacity-0 scale-90'}`}
      style={{ transitionDelay: `${index * 100}ms` }}
    >
      {logo}
      <span>{name}</span>
    </div>
  );
};

export default function Home(): React.ReactElement {
  const [copied, setCopied] = useState(false);
  const [terminalLines, setTerminalLines] = useState<number[]>([]);

  const handleCopy = () => {
    navigator.clipboard.writeText("curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Terminal typing animation
  useEffect(() => {
    const delays = [0, 300, 600, 900, 1200, 1500, 1800];
    delays.forEach((delay, index) => {
      setTimeout(() => {
        setTerminalLines(prev => [...prev, index]);
      }, delay);
    });
  }, []);

  return (
    <Layout
      title="The Open Source Hub for AI Models"
      description="World-class, self-hosted model repository designed for vLLM and SGLang. Secure your AI assets with RBAC, accelerate distribution with proxy caching."
    >
      <main className="bg-[#0d1117] text-slate-300 font-sans selection:bg-green-500/30 selection:text-green-200 w-full overflow-x-hidden">

        {/* Hero Section */}
        <section className="relative pt-20 pb-16 md:pt-28 md:pb-24 lg:pt-36 lg:pb-32 overflow-hidden">
          {/* Animated Background Element */}
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[600px] md:w-[800px] lg:w-[1000px] h-[300px] md:h-[400px] lg:h-[500px] bg-green-500/10 blur-[100px] lg:blur-[120px] rounded-full pointer-events-none animate-pulse"></div>

          {/* Floating particles */}
          <div className="absolute inset-0 overflow-hidden pointer-events-none">
            <div className="absolute top-1/4 left-1/4 w-2 h-2 bg-green-500/30 rounded-full animate-float" style={{ animationDelay: '0s' }}></div>
            <div className="absolute top-1/3 right-1/4 w-3 h-3 bg-emerald-500/20 rounded-full animate-float" style={{ animationDelay: '1s' }}></div>
            <div className="absolute bottom-1/3 left-1/3 w-2 h-2 bg-green-400/25 rounded-full animate-float" style={{ animationDelay: '2s' }}></div>
          </div>

          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-12 items-center">
              {/* Left: Text Content */}
              <div className="text-center lg:text-left">
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-green-900/20 border border-green-800/50 text-green-400 text-xs font-semibold uppercase tracking-wider mb-6 animate-fade-in-up">
                  <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></span>
                  Coming Soon
                </div>
                <h1 className="text-4xl sm:text-5xl lg:text-6xl xl:text-7xl font-extrabold text-white leading-tight mb-6 animate-fade-in-up" style={{ animationDelay: '0.1s' }}>
                  The Open Source <br className="hidden sm:block" />
                  <span className="text-transparent bg-clip-text bg-gradient-to-r from-green-400 to-emerald-600 animate-glow inline-block">Hub</span> for AI Models
                </h1>
                <p className="text-base sm:text-lg text-slate-400 mb-8 max-w-xl mx-auto lg:mx-0 leading-relaxed animate-fade-in-up" style={{ animationDelay: '0.2s' }}>
                  MatrixHub is an open-source, self-hosted AI model registry engineered for large-scale enterprise inference. It serves as a drop-in private replacement for <span className="text-white">Hugging Face</span>, purpose-built to accelerate <span className="text-white">vLLM</span> and <span className="text-white">SGLang</span> workloads.
                </p>

                <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start animate-fade-in-up" style={{ animationDelay: '0.3s' }}>
                  <a
                    href="/blog/deepseek-v4-distribution"
                    className="px-6 sm:px-8 py-3 sm:py-4 bg-green-600 hover:bg-green-500 text-black rounded-lg font-bold text-base sm:text-lg transition-all shadow-[0_0_20px_rgba(22,163,74,0.4)] hover:shadow-[0_0_30px_rgba(22,163,74,0.6)] flex items-center justify-center gap-2 hover:text-black hover:no-underline hover:scale-105 transform"
                  >
                    Quick Start <ArrowRight size={20} />
                  </a>
                  <a
                    href="https://github.com/matrixhub-ai/matrixhub"
                    className="px-6 sm:px-8 py-3 sm:py-4 bg-[#161b22] hover:bg-[#21262d] text-white border border-slate-700 rounded-lg font-bold text-base sm:text-lg transition-all flex items-center justify-center gap-2 hover:text-white hover:no-underline hover:border-slate-600"
                  >
                    <Github size={20} /> View on GitHub
                  </a>
                </div>
              </div>

              {/* Right: Terminal / Visual */}
              <div className="relative group mt-8 lg:mt-0 animate-fade-in-right" style={{ animationDelay: '0.4s' }}>
                <div className="absolute -inset-1 bg-gradient-to-r from-green-600 to-emerald-600 rounded-xl blur opacity-25 group-hover:opacity-40 transition duration-1000"></div>
                <div className="relative rounded-xl bg-[#0d1117] border border-slate-700 shadow-2xl overflow-hidden">
                  <div className="flex items-center gap-2 px-4 py-3 bg-[#161b22] border-b border-slate-800">
                    <div className="w-3 h-3 rounded-full bg-red-500 hover:bg-red-400 transition-colors cursor-pointer"></div>
                    <div className="w-3 h-3 rounded-full bg-yellow-500 hover:bg-yellow-400 transition-colors cursor-pointer"></div>
                    <div className="w-3 h-3 rounded-full bg-green-500 hover:bg-green-400 transition-colors cursor-pointer"></div>
                    <span className="ml-2 text-xs text-slate-500 font-mono">matrixhub-ai/matrixhub</span>
                  </div>
                  <div className="p-4 sm:p-6 font-mono text-xs sm:text-sm overflow-hidden min-h-[240px]">
                    {terminalLines.includes(0) && (
                      <div className="flex items-start text-slate-400 mb-2 animate-fade-in-left">
                        <span className="text-green-500 mr-2 flex-shrink-0">➜</span>
                        <span className="break-all">~ export HF_ENDPOINT=https://hub.matrix.internal</span>
                      </div>
                    )}
                    {terminalLines.includes(1) && (
                      <div className="flex items-start text-slate-400 mb-2 animate-fade-in-left">
                        <span className="text-green-500 mr-2 flex-shrink-0">➜</span>
                        <span className="break-all">~ vllm serve "deepseek-ai/deepseek-coder-33b"</span>
                      </div>
                    )}
                    {terminalLines.includes(2) && (
                      <div className="mt-4 text-slate-300 space-y-1">
                        {terminalLines.includes(3) && (
                          <div className="animate-fade-in-left"><span className="text-blue-400">INFO</span>: Connected to MatrixHub Proxy</div>
                        )}
                        {terminalLines.includes(4) && (
                          <div className="animate-fade-in-left"><span className="text-blue-400">INFO</span>: Cache HIT for layer 1 (sha256:a1b2...)</div>
                        )}
                        {terminalLines.includes(5) && (
                          <div className="animate-fade-in-left"><span className="text-blue-400">INFO</span>: Cache HIT for layer 2 (sha256:c3d4...)</div>
                        )}
                        {terminalLines.includes(6) && (
                          <div className="animate-fade-in-left"><span className="text-green-400">SUCCESS</span>: Model loaded in 2.4s (via LAN)</div>
                        )}
                      </div>
                    )}
                    <div className="mt-4 flex items-center gap-2">
                       <span className="text-green-500 animate-pulse">➜</span>
                       <span className="w-2 h-5 bg-slate-500 terminal-cursor"></span>
                    </div>
                  </div>
                </div>

                {/* Float Card - Hidden on mobile */}
                <div className="absolute -bottom-6 -right-6 bg-[#1c2128] p-4 rounded-lg border border-slate-700 shadow-xl hidden lg:block animate-float">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-green-900/30 rounded-full text-green-400 animate-glow flex items-center justify-center">
                       <Zap size={20} />
                    </div>
                    <div>
                      <div className="text-xs text-slate-400">Download Speed</div>
                      <div className="text-lg font-bold text-white">25.8 GB/s <span className="text-xs font-normal text-green-500">Intranet</span></div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Analogy Section */}
        <AnimatedSection>
          <section className="py-12 sm:py-16 border-y border-slate-800 bg-[#161b22]/50">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
                <h2 className="text-xl sm:text-2xl font-medium text-slate-300 leading-relaxed">
                  <span className="text-transparent bg-clip-text bg-gradient-to-r from-green-400 to-emerald-600 font-extrabold text-2xl sm:text-3xl">MatrixHub</span> is to <span className="text-transparent bg-clip-text bg-gradient-to-r from-yellow-400 to-orange-500 font-extrabold text-2xl sm:text-3xl">Hugging Face</span> what <span className="text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-500 font-extrabold text-2xl sm:text-3xl">Harbor</span> is to <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-indigo-500 font-extrabold text-2xl sm:text-3xl">Docker Hub</span>.
                </h2>
               <p className="mt-4 text-slate-400 text-sm sm:text-base">
                 Stop relying on public internet for mission-critical AI. Control your assets, accelerate your pipelines.
               </p>
            </div>
          </section>
        </AnimatedSection>

        {/* Key Features */}
        <section id="features" className="py-16 sm:py-20 lg:py-24 bg-[#0d1117] relative">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <AnimatedSection className="text-center mb-12 sm:mb-16">
              <h2 className="text-green-500 font-bold tracking-wider uppercase text-xs sm:text-sm mb-2">Core Features</h2>
              <h3 className="text-2xl sm:text-3xl md:text-4xl font-bold text-white mb-4">Infrastructure designed for Scale</h3>
              <p className="text-slate-400 max-w-2xl mx-auto text-sm sm:text-base">
                Built for the specific needs of SREs and Algorithm Engineers managing massive model weights.
              </p>
            </AnimatedSection>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
              <FeatureCard
                icon={<Network size={24} />}
                title="Transparent HF Proxy"
                desc="Drop-in replacement for Hugging Face. Point your HF_ENDPOINT to MatrixHub and keep all training/inference code unchanged."
                index={0}
              />
              <FeatureCard
                icon={<Zap size={24} />}
                title="On-Demand Caching"
                desc="Pull once, cache forever. Automatically localizes public models to slash redundant traffic and accelerate cluster-wide distribution."
                index={1}
              />
              <FeatureCard
                icon={<Shield size={24} />}
                title="RBAC & Audit Logs"
                desc="Fine-grained permissions, project-based isolation, and comprehensive audit trails for every upload and download."
                index={2}
              />
              <FeatureCard
                icon={<Database size={24} />}
                title="Storage Agnostic"
                desc="Compatible with local filesystems, NFS, and S3-compatible backends (MinIO, AWS). Scale to unlimited model capacity."
                index={3}
              />
            </div>
          </div>
        </section>

        {/* Use Cases */}
        <section id="use-cases" className="py-16 sm:py-20 lg:py-24 bg-[#161b22]">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <AnimatedSection className="flex flex-col sm:flex-row justify-between items-start sm:items-end mb-8 sm:mb-12 gap-4">
              <div>
                <h2 className="text-2xl sm:text-3xl font-bold text-white mb-2">Key Use Cases</h2>
                <p className="text-slate-400 text-sm sm:text-base">How organizations use MatrixHub in production.</p>
              </div>
              <a href="#" className="text-green-400 hover:text-green-300 font-medium flex items-center gap-1 hover:no-underline text-sm sm:text-base group">
                View Architecture <ArrowRight size={16} className="group-hover:translate-x-1 transition-transform" />
              </a>
            </AnimatedSection>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 sm:gap-8">
              <UseCaseCard
                icon={<Rocket size={20} />}
                title="Zero-Wait Distribution"
                desc="Eliminate bandwidth bottlenecks with a 'Pull-once, serve-all' cache. Achieve 10Gbps+ speeds across 100+ GPU nodes simultaneously."
                index={0}
              />
              <UseCaseCard
                icon={<ShieldCheck size={20} />}
                title="Air-Gapped Delivery"
                desc="Securely ferry models into isolated networks with integrity protection, malware scanning, and comprehensive audit trails."
                index={1}
              />
              <UseCaseCard
                icon={<Tags size={20} />}
                title="Private Registry"
                desc="Centralize fine-tuned weights with tag locking and CI/CD integration. Guarantee consistency from development to production."
                index={2}
              />
              <UseCaseCard
                icon={<Globe size={20} />}
                title="Global Multi-Region Sync"
                desc="Automate asynchronous, resumable replication between data centers for high availability and low-latency local access."
                index={3}
              />
            </div>
          </div>
        </section>

        {/* Tech Stack Integration */}
        <section className="py-16 sm:py-20 border-t border-slate-800 bg-[#0d1117]">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
            <AnimatedSection>
              <p className="text-slate-500 mb-6 sm:mb-8 font-medium text-xs sm:text-sm tracking-wider">SEAMLESSLY INTEGRATED WITH</p>
            </AnimatedSection>
            <div className="flex flex-wrap justify-center gap-6 sm:gap-8 lg:gap-12">
              <TechLogo logo={<img src="/img/integrations/vllm.png" alt="vLLM" className="w-12 h-12 sm:w-16 sm:h-16 rounded-full" />} name="vLLM" index={0} />
              <TechLogo logo={<img src="/img/integrations/sglang.png" alt="SGLang" className="w-12 h-12 sm:w-16 sm:h-16 rounded-full" />} name="SGLang" index={1} />
              <TechLogo logo={<img src="/img/integrations/kubernetes.png" alt="Kubernetes" className="w-12 h-12 sm:w-16 sm:h-16" />} name="Kubernetes" index={2} />
            </div>
          </div>
        </section>

        {/* CTA Section */}
        <section className="py-16 sm:py-20 lg:py-24 relative overflow-hidden bg-[#0d1117]">
          <div className="absolute inset-0 bg-green-900/10"></div>
          {/* Animated background gradient */}
          <div className="absolute inset-0 bg-gradient-to-r from-green-900/0 via-green-900/20 to-green-900/0 animate-pulse"></div>

          <AnimatedSection className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center relative z-10">
            <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-white mb-4 sm:mb-6">Ready to take control of your AI Models?</h2>
            <p className="text-base sm:text-lg lg:text-xl text-slate-400 mb-8 sm:mb-10">
              Deploy MatrixHub in minutes using Docker Compose or Helm. Open source and free for the community.
            </p>
            <div className="flex flex-col sm:flex-row justify-center gap-4">
               <a
                 href="/docs/intro"
                 className="px-6 sm:px-8 py-3 sm:py-4 bg-white text-black hover:bg-slate-200 rounded-lg font-bold text-base sm:text-lg transition-all hover:text-black hover:no-underline hover:scale-105 transform"
               >
                  Read the Docs
               </a>
               <div className="flex items-center bg-[#0d1117] border border-slate-700 rounded-lg p-1 pr-2 sm:pr-4 hover:border-slate-600 transition-colors">
                 <div className="px-3 sm:px-4 py-2 sm:py-3 text-slate-400 font-mono text-xs sm:text-sm truncate max-w-[200px] sm:max-w-none">
                   curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d
                 </div>
                 <button
                  onClick={handleCopy}
                  className="ml-1 sm:ml-2 p-2 hover:bg-slate-800 rounded text-slate-400 hover:text-white transition-colors relative border-none cursor-pointer bg-transparent flex-shrink-0"
                 >
                   {copied ? <span className="text-green-500 font-bold text-xs">Copied!</span> : <span className="text-xs font-bold border border-slate-600 px-2 py-1 rounded hover:border-green-500 transition-colors">COPY</span>}
                 </button>
               </div>
            </div>
          </AnimatedSection>
        </section>
      </main>
    </Layout>
  );
}
